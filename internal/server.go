package internal

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/franz-dalitz/device-sharing-thb/internal/data"
	"github.com/franz-dalitz/device-sharing-thb/internal/notifications"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	db       = data.Database{}
	hub      = notifications.NewHub()
	upgrader = notifications.DefaultUpgrader()
)

func Server() *gin.Engine {
	server := gin.Default()
	db.Mock()

	// static content and page routing
	server.Static("/static/htmx", "web/node_modules/htmx.org/dist")
	server.Static("/static/bootstrap", "web/node_modules/bootstrap/dist")
	server.Static("/static/bootstrap-icons", "web/node_modules/bootstrap-icons/font")
	server.Static("/static/thb.svg", "web/thb.svg")
	server.Static("/static/photos", "photos")

	server.LoadHTMLGlob("web/**/*.tmpl")

	server.GET("/:page", func(c *gin.Context) {
		page := strings.TrimPrefix(c.Param("page"), "/")
		page = page + ".tmpl"
		c.HTML(http.StatusOK, page, nil)
	})

	// API
	server.PUT("/api/suspend/:id", toggleSuspend)
	server.PUT("/api/devices", updateDevice)
	server.GET("/edit", loadEditPage)
	server.GET("/api/contact", contactUser)
	server.PUT("/api/reserve", toggleReserve)
	server.POST("/api/messages", sendMessage)
	server.GET("/api/messages", loadMessages)
	server.POST("/api/devices", createDevice)
	server.DELETE("/api/devices/:id", deleteDevice)
	server.PUT("/api/like", toggleLike)
	server.GET("/ws", registerClient)
	server.GET("/api/search", search)
	server.GET("/api/component/category-list", getCategoryList)
	server.GET("/api/useropts", getUserOpts)
	server.GET("/api/mail", loadMail)
	server.POST("/api/userselect", selectUser)
	server.GET("/api/components/contacts", getContacts)

	go hub.Run()

	return server
}

// @Description toggle suspension for a device
// @Param id path int true "device id"
// @Success 200
// @Router /api/suspend/{id} [put]
func toggleSuspend(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		slog.Error(err.Error())
		return
	}
	ix := slices.IndexFunc(db.Devices, func(dbDevice *data.Device) bool {
		return dbDevice.ID == id
	})
	if ix == -1 {
		slog.Error("trying to toggle suspend on nonexistent device")
		return
	}
	device := db.Devices[ix]
	if device.Suspended {
		device.Suspended = false
		hub.Deliver <- &notifications.Event{
			Recipient: device.Owner,
			Content:   "You un-suspended \"" + device.Title + "\"",
		}
	} else {
		device.Suspended = true
		hub.Deliver <- &notifications.Event{
			Recipient: device.Owner,
			Content:   "You suspended \"" + device.Title + "\"",
		}
	}
}

// @Description update an existing device
// @Param userID formData int true "owner id"
// @Param deviceID formData int true "device id"
// @Param title formData string true "device title"
// @Param category formData string true "device category"
// @Param description formData string true "device description"
// @Param location formData string true "device location"
// @Param photo formData file false "device photo"
// @Success 200
// @Router /api/devices [put]
func updateDevice(c *gin.Context) {
	var req struct {
		Owner       int    `form:"userID"`
		ID          int    `form:"deviceID"`
		Title       string `form:"title"`
		Category    string `form:"category"`
		Description string `form:"description"`
		Location    string `form:"location"`
	}
	if err := c.ShouldBind(&req); err != nil {
		slog.Error(err.Error())
		return
	}
	photo, header, err := c.Request.FormFile("photo")
	if err != nil {
		slog.Error(err.Error())
	}
	if err := c.ShouldBind(&req); err != nil {
		slog.Error(err.Error())
		return
	}
	uIx := slices.IndexFunc(db.Users, func(dbUser *data.User) bool {
		return dbUser.ID == req.Owner
	})
	if uIx == -1 {
		slog.Error("trying to edit device with nonexistent user")
		return
	}
	dIx := slices.IndexFunc(db.Devices, func(dbDevice *data.Device) bool {
		return dbDevice.ID == req.ID
	})
	if dIx == -1 {
		slog.Error("trying to update nonexistent device")
		return
	}
	cat, err := data.AsCategory(req.Category)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	device := db.Devices[dIx]
	if req.Owner != device.Owner {
		slog.Error("user is trying to update device they don't own")
		return
	}
	if photo != nil {
		defer photo.Close()
		uu, err := uuid.NewRandom()
		if err != nil {
			slog.Error(err.Error())
			return
		}
		filename := uu.String() + filepath.Ext(header.Filename)
		filePath := filepath.Join("photos", filename)
		dst, err := os.Create(filePath)
		if err != nil {
			slog.Error(err.Error())
			return
		}
		defer dst.Close()
		if _, err = io.Copy(dst, photo); err != nil {
			slog.Error(err.Error())
			return
		}
		if device.Photo != "" {
			err = os.Remove("photos/" + device.Photo)
			if err != nil {
				slog.Error(err.Error())
				return
			}
		}
		device.Photo = filename
	}
	device.Title = req.Title
	device.Category = cat
	device.Description = req.Description
	device.Location = req.Location
	c.Header("HX-Redirect", "/offers")
	c.Status(http.StatusOK)
}

// @Description load the edit page for a specific device
// @Param id query int true "device id"
// @Success 200
// @Router /edit [get]
func loadEditPage(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("device"))
	if err != nil {
		slog.Error(err.Error())
		return
	}
	dIx := slices.IndexFunc(db.Devices, func(dbDevice *data.Device) bool {
		return dbDevice.ID == id
	})
	if dIx == -1 {
		slog.Error("trying to edit nonexistent device")
		return
	}
	c.HTML(http.StatusOK, "edit", db.Devices[dIx])
}

// @Description delete a device
// @Param id path int true "device id"
// @Success 200
// @Router /api/devices/{id} [delete]
func deleteDevice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		slog.Error(err.Error())
		return
	}
	dIx := slices.IndexFunc(db.Devices, func(dbDevice *data.Device) bool {
		return dbDevice.ID == id
	})
	if dIx == -1 {
		slog.Error("trying to delete nonexistent device")
		return
	}
	err = os.Remove("photos/" + db.Devices[dIx].Photo)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	db.Devices = slices.Delete(db.Devices, dIx, dIx+1)
	c.Header("HX-Refresh", "true")
	c.Status(http.StatusOK)
}

// @Description create contact with user if not present, then redirect to chat
// @Param userID query int true "user id"
// @Param otherID query int true "id of a different user"
// @Success 200
// @Router /api/contact [get]
func contactUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Query("userID"))
	if err != nil {
		slog.Error(err.Error())
		return
	}
	otherID, err := strconv.Atoi(c.Query("otherID"))
	if err != nil {
		slog.Error(err.Error())
		return
	}
	if userID == otherID {
		slog.Error("can't create chat between user and themselves")
		return
	}
	uIx := slices.IndexFunc(db.Users, func(dbUser *data.User) bool {
		return dbUser.ID == userID
	})
	if uIx == -1 {
		slog.Error("trying to access nonexistent user")
		return
	}
	oIx := slices.IndexFunc(db.Users, func(dbUser *data.User) bool {
		return dbUser.ID == otherID
	})
	if oIx == -1 {
		slog.Error("trying to access nonexistent user")
		return
	}
	other := db.Users[oIx]
	pair := data.IntPair{X: userID, Y: otherID}
	cIx := slices.IndexFunc(db.Chats, func(dbChat *data.Chat) bool {
		return dbChat.Participants.Equals(pair)
	})
	var chatID int
	if cIx == -1 {
		newChat := data.NewChat(pair)
		db.Chats = append(db.Chats, newChat)
		chatID = newChat.ID
	} else {
		chatID = db.Chats[cIx].ID
	}
	c.Header("HX-Redirect", fmt.Sprintf("/chat?chat=%v&user=%v", chatID, other.Name))
	c.Status(http.StatusOK)
}

// @Description toggle reservation for a device
// @Param userID formData int true "user id"
// @Param deviceID formData int true "device id"
// @Success 200
// @Router /api/reserve [put]
func toggleReserve(c *gin.Context) {
	var req struct {
		UserID   int `form:"userID"`
		DeviceID int `form:"deviceID"`
	}
	err := c.ShouldBind(&req)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	uIx := slices.IndexFunc(db.Users, func(dbUser *data.User) bool {
		return dbUser.ID == req.UserID
	})
	if uIx == -1 {
		slog.Error("trying to access nonexistent user")
		return
	}
	dIx := slices.IndexFunc(db.Devices, func(dbDevice *data.Device) bool {
		return dbDevice.ID == req.DeviceID
	})
	if dIx == -1 {
		slog.Error("trying to access nonexistent device")
		return
	}
	device := db.Devices[dIx]
	if device.ReservedBy != -1 && device.ReservedBy != req.UserID {
		slog.Error("user trying to toggle reserve for device reserved by other user")
		return
	} else if device.ReservedBy == -1 {
		device.ReservedBy = req.UserID
		hub.Deliver <- &notifications.Event{
			Recipient: req.UserID,
			Content:   "You reserved \"" + device.Title + "\"",
		}
	} else {
		device.ReservedBy = -1
		hub.Deliver <- &notifications.Event{
			Recipient: req.UserID,
			Content:   "You un-reserved \"" + device.Title + "\"",
		}
	}
}

// @Description send a message in a chat
// @Param userID formData int true "sender id"
// @Param chatID formData int true "chat id"
// @Param message formData string true "message content"
// @Success 200
// @Router /api/messages [post]
func sendMessage(c *gin.Context) {
	var req struct {
		UserID  int    `form:"userID"`
		ChatID  int    `form:"chatID"`
		Message string `form:"message"`
	}
	err := c.ShouldBind(&req)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	uIx := slices.IndexFunc(db.Users, func(dbUser *data.User) bool {
		return dbUser.ID == req.UserID
	})
	if uIx == -1 {
		slog.Error("trying to access nonexistent user")
		return
	}
	user := db.Users[uIx]
	cIx := slices.IndexFunc(db.Chats, func(dbChat *data.Chat) bool {
		return dbChat.ID == req.ChatID
	})
	if cIx == -1 {
		slog.Error("trying to access nonexistent chat")
		return
	}
	chat := db.Chats[cIx]
	if !chat.Participants.Contains(req.UserID) {
		slog.Error("not allowed to acces this chat as this user")
		return
	}
	tmpl, err := template.ParseFiles("web/components/messages.tmpl")
	if err != nil {
		slog.Error(err.Error())
		return
	}
	var content bytes.Buffer
	err = tmpl.Execute(&content, []struct {
		Self    bool
		Content string
	}{
		{
			Self:    true,
			Content: req.Message,
		},
	})
	if err != nil {
		slog.Error(err.Error())
		return
	}
	other, err := chat.Participants.Other(req.UserID)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	chat.Messages = append(chat.Messages, data.NewMessage(req.UserID, req.Message))
	hub.Deliver <- &notifications.Event{
		Type:      notifications.Notification,
		Recipient: other,
		Content:   "<a href=\"chat?chat=" + strconv.Itoa(chat.ID) + "&user=" + user.Name + "\">" + user.Name + " sent you a message!</a>",
	}
	hub.Deliver <- &notifications.Event{
		Type:      notifications.Message,
		Recipient: other,
		Content:   req.Message,
	}
	c.Data(http.StatusOK, "text/html", content.Bytes())
}

// @Description load all messages in a chat
// @Param userID query int true "user id"
// @Param chatID query int true "chat id"
// @Success 200
// @Router /api/messages [get]
func loadMessages(c *gin.Context) {
	userID, err := strconv.Atoi(c.Query("userID"))
	if err != nil {
		slog.Error(err.Error())
		return
	}
	chatID, err := strconv.Atoi(c.Query("chatID"))
	if err != nil {
		slog.Error(err.Error())
		return
	}
	if slices.IndexFunc(db.Users, func(dbUser *data.User) bool {
		return dbUser.ID == userID
	}) == -1 {
		slog.Error("trying to access nonexistent user")
		return
	}
	cIx := slices.IndexFunc(db.Chats, func(dbChat *data.Chat) bool {
		return dbChat.ID == chatID
	})
	if cIx == -1 {
		slog.Error("trying to access nonexistent chat")
		return
	}
	chat := db.Chats[cIx]
	if !chat.Participants.Contains(userID) {
		slog.Error("not allowed to acces this chat as this user")
		return
	}
	tmpl, err := template.ParseFiles("web/components/messages.tmpl")
	if err != nil {
		slog.Error(err.Error())
		return
	}
	type msg struct {
		Self    bool
		Content string
	}
	messages := []*msg{}
	for _, chatMsg := range chat.Messages {
		messages = append(messages, &msg{
			Self:    userID == chatMsg.By,
			Content: chatMsg.Content,
		})
	}
	var content bytes.Buffer
	err = tmpl.Execute(&content, messages)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	c.Data(http.StatusOK, "text/html", content.Bytes())
}

// @Description get all contacts with a user
// @Param userID query int true "user id"
// @Success 200
// @Router /api/components/contacts [get]
func getContacts(c *gin.Context) {
	userID, err := strconv.Atoi(c.Query("userID"))
	if err != nil {
		slog.Error(err.Error())
		return
	}
	userIndex := slices.IndexFunc(db.Users, func(dbUser *data.User) bool {
		return dbUser.ID == userID
	})
	if userIndex == -1 {
		slog.Error("trying to get contacts of nonexistent user")
		return
	}
	tmpl, err := template.ParseFiles("web/components/contacts.tmpl")
	if err != nil {
		slog.Error(err.Error(), "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	type contact struct {
		ChatID int
		Name   string
		Online bool
		Seen   string
	}
	contacts := []*contact{}
	for _, chat := range db.Chats {
		if chat.Participants.Contains(userID) {
			otherID, err := chat.Participants.Other(userID)
			if err != nil {
				slog.Error(err.Error())
				return
			}
			otherIndex := slices.IndexFunc(db.Users, func(dbUser *data.User) bool {
				return dbUser.ID == otherID
			})
			if otherIndex == -1 {
				slog.Error("trying to access nonexistent user")
				return
			}
			other := db.Users[otherIndex]
			since := time.Since(other.Seen)
			var seen string
			if since < time.Hour {
				seen = strconv.Itoa(int(since.Minutes())) + " Minute(s)"
			} else if since < 24*time.Hour {
				seen = strconv.Itoa(int(since.Hours())) + " Hour(s)"
			} else {
				seen = strconv.Itoa(int(since.Hours()/24)) + " Day(s)"
			}
			contacts = append(contacts, &contact{
				ChatID: chat.ID,
				Name:   other.Name,
				Online: time.Since(other.Seen) < time.Minute,
				Seen:   seen,
			})
		}
	}
	var content bytes.Buffer
	err = tmpl.Execute(&content, contacts)
	if err != nil {
		slog.Error(err.Error(), "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, "text/html", content.Bytes())
}

// @Description toggle like of user for device
// @Param userID formData int true "user id"
// @Param deviceID formData int true "device id"
// @Success 200
// @Router /api/like [put]
func toggleLike(c *gin.Context) {
	var req struct {
		UserID   int `form:"userID"`
		DeviceID int `form:"deviceID"`
	}
	if err := c.ShouldBind(&req); err != nil {
		slog.Error(err.Error())
		return
	}
	uIx := slices.IndexFunc(db.Users, func(dbUser *data.User) bool {
		return dbUser.ID == req.UserID
	})
	if uIx == -1 {
		slog.Error("trying to toggle like with nonexistent user")
		return
	}
	dIx := slices.IndexFunc(db.Devices, func(dbDevice *data.Device) bool {
		return dbDevice.ID == req.DeviceID
	})
	if dIx == -1 {
		slog.Error("trying to toggle like for nonexistent device")
		return
	}
	user := db.Users[uIx]
	udIx := slices.Index(user.Liked, req.DeviceID)
	if udIx == -1 {
		user.Liked = append(user.Liked, req.DeviceID)
		hub.Deliver <- &notifications.Event{
			Type:      notifications.Notification,
			Recipient: db.Devices[dIx].Owner,
			Content:   "\"" + db.Devices[dIx].Title + "\" wurde geliked!",
		}
		hub.Deliver <- &notifications.Event{
			Recipient: req.UserID,
			Content:   "You liked \"" + db.Devices[dIx].Title + "\"",
		}
	} else {
		user.Liked = slices.Delete(user.Liked, udIx, udIx+1)
		hub.Deliver <- &notifications.Event{
			Recipient: req.UserID,
			Content:   "You un-liked \"" + db.Devices[dIx].Title + "\"",
		}
	}
}

// @Description establish a notification client websocket connection
// @Param userID query int true "user id"
// @Success 200
// @Router /ws [get]
func registerClient(c *gin.Context) {
	uid, err := strconv.Atoi(c.Query("userID"))
	if err != nil {
		slog.Error(err.Error())
		return
	}
	uIx := slices.IndexFunc(db.Users, func(dbUser *data.User) bool {
		return dbUser.ID == uid
	})
	if uIx == -1 {
		slog.Error("trying to register client for nonexistent user")
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	client := notifications.NewClient(hub, conn, db.Users[uIx])
	hub.Register <- client
	client.Run()
}

// @Description get dropdown list options for list of all categories
// @Param preselect query bool false "whether to select the active category of a device"
// @Param deviceID query int false "the device for which to pre-select"
// @Success 200
// @Router /api/component/category-list [get]
func getCategoryList(c *gin.Context) {
	tmpl, err := template.ParseFiles("web/components/category-list.tmpl")
	if err != nil {
		slog.Error(err.Error(), "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	type ListCategory struct {
		Name     string
		Selected bool
	}
	dCat := -1
	if c.Query("preselect") == "true" {
		dId, err := strconv.Atoi(c.Query("deviceID"))
		if err != nil {
			slog.Error(err.Error())
			return
		}
		dIx := slices.IndexFunc(db.Devices, func(dbDevice *data.Device) bool {
			return dbDevice.ID == dId
		})
		if dIx == -1 {
			slog.Error("trying to edit nonexistent device")
			return
		}
		device := db.Devices[dIx]
		dCat = int(device.Category)
	}
	cats := []ListCategory{}
	for _, cat := range data.Categories {
		catVal, err := data.AsCategory(cat)
		if err != nil {
			slog.Error(err.Error())
			return
		}
		cats = append(cats, ListCategory{
			Name:     cat,
			Selected: dCat == int(catVal),
		})
	}

	var content bytes.Buffer
	err = tmpl.Execute(&content, cats)
	if err != nil {
		slog.Error(err.Error(), "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, "text/html", content.Bytes())
}

// @Description return device cards for specific search query
// @Param userID query int true "id of searching user"
// @Param category query string true "device category"
// @Param search query string false "search string"
// @Param uExcept query bool false "whether to only return devices OTHER than the user"
// @Param uOnly query bool false "whether to only return devices BY that user"
// @Param offers query bool false "whether the user is trying to look at offers, rather than devices"
// @Param searchType query string false "what kind of query we're making (browse, liked, reserved, suspended)"
// @Success 200
// @Router /api/search [get]
func search(c *gin.Context) {
	search := c.Query("search")
	category := c.Query("category")
	uExcept := c.Query("uExcept")
	uOnly := c.Query("uOnly")
	offers := c.Query("offers")
	searchType := c.Query("searchType")
	userID, err := strconv.Atoi(c.Query("userID"))
	if err != nil {
		return
	}
	uIx := slices.IndexFunc(db.Users, func(dbUser *data.User) bool {
		return dbUser.ID == userID
	})
	if uIx == -1 {
		return
	}
	user := db.Users[uIx]
	filtered := []*data.Device{}
	for _, device := range db.Devices {
		if searchType == "suspended" {
			if !device.Suspended {
				continue
			}
		} else if device.Suspended {
			continue
		}
		if searchType == "reserved" {
			if device.ReservedBy != user.ID && offers == "false" || device.ReservedBy == -1 && offers == "true" {
				continue
			}
		} else if device.ReservedBy != -1 {
			continue
		}
		if searchType == "liked" {
			if i := slices.Index(user.Liked, device.ID); i == -1 {
				continue
			}
		} else if i := slices.Index(user.Liked, device.ID); i != -1 {
			continue
		}
		if !strings.Contains(strings.ReplaceAll(strings.ToLower(device.Title), " ", ""), strings.ToLower(search)) {
			continue
		}
		if category != "Any" && device.Category.String() != category {
			continue
		}
		if uExcept == "true" {
			if device.Owner == userID {
				continue
			}
		} else if uOnly == "true" {
			if device.Owner != userID {
				continue
			}
		}
		filtered = append(filtered, device)
	}
	type OfferDevice struct {
		Device     *data.Device
		SearchType string
	}
	type SearchDevice struct {
		Device     *data.Device
		OwnerID    int
		OwnerName  string
		ChatID     int
		Likes      int
		Liked      bool
		SearchType string
	}
	if offers == "true" {
		devices := []OfferDevice{}
		for _, device := range filtered {
			devices = append(devices, OfferDevice{
				device,
				searchType,
			})
		}
		c.HTML(http.StatusOK, "offer-cards", devices)
	} else {
		devices := []SearchDevice{}
		for _, device := range filtered {
			oIx := slices.IndexFunc(db.Users, func(dbUser *data.User) bool {
				return dbUser.ID == device.Owner
			})
			if oIx == -1 {
				return
			}
			other := db.Users[oIx]
			cIx := slices.IndexFunc(db.Chats, func(dbChat *data.Chat) bool {
				return dbChat.Participants.Equals(data.IntPair{X: userID, Y: other.ID})
			})
			var chat int
			if cIx == -1 {
				chat = -1
			} else {
				chat = db.Chats[cIx].ID
			}
			likes := 0
			for _, u := range db.Users {
				if i := slices.Index(u.Liked, device.ID); i != -1 {
					likes++
				}
			}
			devices = append(devices, SearchDevice{
				device,
				other.ID,
				other.Name,
				chat,
				likes,
				slices.Index(user.Liked, device.ID) != -1,
				searchType,
			})
		}
		sort.Slice(devices, func(i, j int) bool {
			return devices[i].Likes > devices[j].Likes
		})
		c.HTML(http.StatusOK, "device-cards", devices)
	}
}

// @Description get a list of the selectable mocked users
// @Param userID query int true "..."
// @Success 200
// @Router /api/useropts [get]
func getUserOpts(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("userID"))
	if err != nil {
		slog.Error(err.Error(), "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	tmpl, err := template.ParseFiles("web/components/useropts.tmpl")
	if err != nil {
		slog.Error(err.Error(), "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var content bytes.Buffer
	err = tmpl.Execute(&content, gin.H{
		"Users":    db.Users,
		"Selected": id,
	})
	if err != nil {
		slog.Error(err.Error(), "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, "text/html", content.Bytes())
}

// @Description get the email component of a mocked user
// @Param userID query int true "user id"
// @Success 200
// @Router /api/mail [get]
func loadMail(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("userID"))
	if err != nil {
		slog.Error(err.Error(), "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	tmpl, err := template.ParseFiles("web/components/mail.tmpl")
	if err != nil {
		slog.Error(err.Error(), "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	ix := slices.IndexFunc(db.Users, func(dbUser *data.User) bool {
		return dbUser.ID == id
	})
	if ix == -1 {
		return
	}
	user := db.Users[ix]

	var content bytes.Buffer
	err = tmpl.Execute(&content, user)
	if err != nil {
		slog.Error(err.Error(), "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Data(http.StatusOK, "text/html", content.Bytes())
}

// @Description select a new mocked user
// @Param id formData int true "user id"
// @Success 200
// @Router /api/userselect [post]
func selectUser(c *gin.Context) {
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		slog.Error(err.Error(), "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
	}
	ix := slices.IndexFunc(db.Users, func(dbUser *data.User) bool {
		return dbUser.ID == id
	})
	if ix == -1 {
		return
	}
	user := db.Users[ix]
	tmpl, err := template.ParseFiles("web/components/mail.tmpl")
	if err != nil {
		slog.Error(err.Error(), "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	var content bytes.Buffer
	err = tmpl.Execute(&content, user)
	if err != nil {
		slog.Error(err.Error(), "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, "text/html", content.Bytes())
}

// @Description create a new device
// @Param userID formData int true "user id"
// @Param title formData string true "device title"
// @Param category formData string true "device category"
// @Param description formData string true "device description"
// @Param location formData string true "device location"
// @Param photo formData file false "device photo"
// @Success 200
// @Router /api/devices [post]
func createDevice(c *gin.Context) {
	var req struct {
		Owner       int    `form:"userID"`
		Title       string `form:"title"`
		Category    string `form:"category"`
		Description string `form:"description"`
		Location    string `form:"location"`
	}
	photo, header, err := c.Request.FormFile("photo")
	if err != nil {
		slog.Error(err.Error())
		return
	}
	defer photo.Close()
	if err := c.ShouldBind(&req); err != nil {
		slog.Error(err.Error())
		return
	}
	cat, err := data.AsCategory(req.Category)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	uu, err := uuid.NewRandom()
	if err != nil {
		slog.Error(err.Error())
		return
	}
	filename := uu.String() + filepath.Ext(header.Filename)
	filePath := filepath.Join("photos", filename)
	dst, err := os.Create(filePath)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	defer dst.Close()
	if _, err = io.Copy(dst, photo); err != nil {
		slog.Error(err.Error())
		return
	}
	device := data.NewDevice(req.Owner, req.Title, cat, req.Description, req.Location).WithPhoto(filename)
	db.Devices = append(db.Devices, device)
	hub.Deliver <- &notifications.Event{
		Recipient: device.Owner,
		Content:   "Created offer \"" + device.Title + "\"",
	}
}
