package internal

import (
	"bytes"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/franz-dalitz/device-sharing-thb/internal/data"
	"github.com/franz-dalitz/device-sharing-thb/internal/notifications"
	"github.com/gin-gonic/gin"
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

	server.LoadHTMLGlob("web/**/*.tmpl")

	server.GET("/:page", func(c *gin.Context) {
		page := strings.TrimPrefix(c.Param("page"), "/")
		page = page + ".tmpl"
		c.HTML(http.StatusOK, page, nil)
	})

	// API
	server.GET("/api/contact", contactUser)
	server.PUT("/api/reserve", toggleReserve)
	server.POST("/api/messages", sendMessage)
	server.GET("/api/messages", loadMessages)
	server.POST("/api/devices", createDevice)
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
	} else {
		device.ReservedBy = -1
	}
}

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
	} else {
		user.Liked = slices.Delete(user.Liked, udIx, udIx+1)
	}
}

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

func getCategoryList(c *gin.Context) {
	tmpl, err := template.ParseFiles("web/components/category-list.tmpl")
	if err != nil {
		slog.Error(err.Error(), "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var content bytes.Buffer
	err = tmpl.Execute(&content, data.Categories)
	if err != nil {
		slog.Error(err.Error(), "err", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, "text/html", content.Bytes())
}

func search(c *gin.Context) {
	search := c.Query("search")
	category := c.Query("category")
	uExcept := c.Query("uExcept")
	uOnly := c.Query("uOnly")
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
	type SearchDevice struct {
		Device    *data.Device
		OwnerID   int
		OwnerName string
		ChatID    int
		Liked     bool
	}
	devices := []SearchDevice{}
	for _, device := range db.Devices {
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
		devices = append(devices, SearchDevice{
			device,
			other.ID,
			other.Name,
			chat,
			slices.IndexFunc(user.Liked, func(dId int) bool {
				return dId == device.ID
			}) != -1,
		})
	}
	c.HTML(http.StatusOK, "device-cards", devices)
}

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

func createDevice(c *gin.Context) {
	var req struct {
		Owner       int                  `form:"userID"`
		Title       string               `form:"title"`
		Category    string               `form:"category"`
		Description string               `form:"description"`
		Location    string               `form:"location"`
		Photo       multipart.FileHeader `form:"photo"`
	}

	if err := c.ShouldBind(&req); err != nil {
		slog.Error(err.Error())
		return
	}

	cat, err := data.AsCategory(req.Category)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	device := data.NewDevice(req.Owner, req.Title, cat, req.Description, req.Location).WithPhoto(req.Photo)
	db.Devices = append(db.Devices, device)
}
