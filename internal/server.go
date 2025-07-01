package internal

import (
	"bytes"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/gin-gonic/gin"
)

func Server() *gin.Engine {
	server := gin.Default()

	// static content and page routing
	server.Static("/static/htmx", "web/node_modules/htmx.org/dist")
	server.Static("/static/bootstrap", "web/node_modules/bootstrap/dist")
	server.Static("/static/bootstrap-icons", "web/node_modules/bootstrap-icons/font")

	server.LoadHTMLGlob("web/**/*.tmpl")

	server.GET("/:page", func(c *gin.Context) {
		page := strings.TrimPrefix(c.Param("page"), "/")
		page = page + ".tmpl"
		c.HTML(http.StatusOK, page, nil)
	})

	// // API
	// server.GET("/api/users", listUsers)
	// server.GET("/api/users/:id", getUser)
	// server.GET("/api/devices", listDevices)
	// server.GET("/api/devices/query", queryDevices)
	// server.GET("/api/devices/:id", getDevice)
	// server.POST("/api/devices", createDevice)
	// server.DELETE("/api/devices/:id", deleteDevice)
	// server.PUT("/api/devices", updateDevice)
	// server.GET("/api/categories", listCategories)
	// server.PUT("/api/like/:user/:device", toggleLike)
	// server.GET("/api/chats/u/:user", listChats)
	// server.GET("/api/chats/:id", getChat)
	// server.POST("/api/chats", createChat)
	// server.POST("/api/chats/:id/messages", createMessage)

	// server.GET("/api/search", func(c *gin.Context) {
	// 	search := c.Query("search")

	// 	devices := []*Device{}

	// 	for _, device := range Db.Devices {
	// 		if strings.Contains(strings.ReplaceAll(strings.ToLower(device.Name), " ", ""), strings.ToLower(search)) {
	// 			devices = append(devices, device)
	// 		}
	// 	}

	// 	c.HTML(http.StatusOK, "device-cards", gin.H{
	// 		"Devices": devices,
	// 	})
	// })

	server.GET("/api/useropts", getUserOpts)
	server.GET("/api/mail", loadMail)
	server.POST("/api/userselect", selectUser)

	return server
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
		"Users":    Db.Users,
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

	ix := slices.IndexFunc(Db.Users, func(dbUser *User) bool {
		return dbUser.ID == id
	})
	if ix == -1 {
		return
	}
	user := Db.Users[ix]

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
	ix := slices.IndexFunc(Db.Users, func(dbUser *User) bool {
		return dbUser.ID == id
	})
	if ix == -1 {
		return
	}
	user := Db.Users[ix]

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

// func listUsers(c *gin.Context) {
// 	var uIds []int
// 	for _, user := range Db.Users {
// 		uIds = append(uIds, user.ID)
// 	}

// 	c.JSON(http.StatusOK, uIds)
// }

// func getUser(c *gin.Context) {
// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return
// 	}

// 	ix := slices.IndexFunc(Db.Users, func(dbUser *User) bool {
// 		return dbUser.ID == id
// 	})

// 	if ix == -1 {
// 		slog.Error("trying to get a nonexistent user")
// 		return
// 	}

// 	c.JSON(http.StatusOK, Db.Users[ix])
// }

// func listDevices(c *gin.Context) {
// 	var dIds []int
// 	for _, device := range Db.Devices {
// 		dIds = append(dIds, device.ID)
// 	}

// 	c.JSON(http.StatusOK, dIds)
// }

// type DeviceQuery struct {
// 	Search   string   `json:"search"`
// 	Category Category `json:"category"`
// }

// func queryDevices(c *gin.Context) {
// 	var query DeviceQuery
// 	if err := c.BindJSON(&query); err != nil {
// 		slog.Error("failed to parse device query", "error", err)
// 		return
// 	}

// 	devices := []*Device{}

// 	for _, device := range Db.Devices {
// 		if !strings.Contains(strings.ReplaceAll(strings.ToLower(device.Name), " ", ""), strings.ToLower(query.Search)) {
// 			continue
// 		}

// 		if query.Category != 0 && device.Category != query.Category {
// 			continue
// 		}

// 		devices = append(devices, device)
// 	}

// 	c.JSON(http.StatusOK, devices)
// }

// func getDevice(c *gin.Context) {
// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return
// 	}

// 	ix := slices.IndexFunc(Db.Devices, func(dbDevice *Device) bool {
// 		return dbDevice.ID == id
// 	})

// 	if ix == -1 {
// 		slog.Error("trying to get a nonexistent device")
// 		return
// 	}

// 	c.JSON(http.StatusOK, Db.Devices[id])
// }

// func createDevice(c *gin.Context) {
// 	var device Device
// 	if err := c.BindJSON(&device); err != nil {
// 		slog.Error(err.Error())
// 		return
// 	}

// 	Db.Devices = append(Db.Devices, &device)
// }

// func deleteDevice(c *gin.Context) {
// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return
// 	}

// 	Db.Devices = slices.DeleteFunc(Db.Devices, func(device *Device) bool {
// 		return device.ID == id
// 	})

// 	for _, user := range Db.Users {
// 		user.Liked = slices.DeleteFunc(user.Liked, func(dbId int) bool {
// 			return dbId == id
// 		})
// 	}
// }

// func updateDevice(c *gin.Context) {
// 	var device Device
// 	if err := c.BindJSON(&device); err != nil {
// 		slog.Error(err.Error())
// 		return
// 	}

// 	i := slices.IndexFunc(Db.Devices, func(DbDevice *Device) bool {
// 		return DbDevice.ID == device.ID
// 	})

// 	if i == -1 {
// 		slog.Error("trying to update nonexistent device")
// 		return
// 	}

// 	Db.Devices[i] = &device
// }

// func listCategories(c *gin.Context) {
// 	c.JSON(http.StatusOK, Categories)
// }

// func toggleLike(c *gin.Context) {
// 	uId, err := strconv.Atoi(c.Param("user"))
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return
// 	}

// 	dId, err := strconv.Atoi(c.Param("device"))
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return
// 	}

// 	uIx := slices.IndexFunc(Db.Users, func(dbUser *User) bool {
// 		return dbUser.ID == uId
// 	})

// 	if uIx == -1 {
// 		slog.Error("trying to toggle like with nonexistent user")
// 		return
// 	}

// 	dIx := slices.IndexFunc(Db.Devices, func(dbDevice *Device) bool {
// 		return dbDevice.ID == dId
// 	})

// 	if dIx == -1 {
// 		slog.Error("trying to toggle like for nonexistent device")
// 		return
// 	}

// 	user := Db.Users[uIx]
// 	udIx := slices.Index(user.Liked, dId)

// 	if udIx == -1 {
// 		user.Liked = append(user.Liked, dId)
// 	} else {
// 		user.Liked = slices.Delete(user.Liked, udIx, udIx)
// 	}
// }

// func listChats(c *gin.Context) {
// 	uId, err := strconv.Atoi(c.Param("user"))
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return
// 	}

// 	uIx := slices.IndexFunc(Db.Users, func(dbUser *User) bool {
// 		return dbUser.ID == uId
// 	})

// 	if uIx == -1 {
// 		slog.Error("trying to list chats of nonexistent user")
// 		return
// 	}

// 	var chats []int
// 	for _, chat := range Db.Chats {
// 		if chat.Participants.Contains(uId) {
// 			chats = append(chats, chat.ID)
// 		}
// 	}

// 	c.JSON(http.StatusOK, chats)
// }

// func getChat(c *gin.Context) {
// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return
// 	}

// 	ix := slices.IndexFunc(Db.Chats, func(dbChat *Chat) bool {
// 		return dbChat.ID == id
// 	})

// 	if ix == -1 {
// 		slog.Error("trying to get nonexistent chat")
// 		return
// 	}

// 	c.JSON(http.StatusOK, Db.Chats[ix])
// }

// func createChat(c *gin.Context) {
// 	var pair IntPair
// 	if err := c.BindJSON(&pair); err != nil {
// 		slog.Error("failed to bind participant pair", "error", err)
// 		return
// 	}

// 	if pair.X == pair.Y {
// 		slog.Error("can't create chat between user and themselves")
// 		return
// 	}

// 	for _, chat := range Db.Chats {
// 		if chat.Participants.Equals(pair) {
// 			slog.Error("a chat between these users already exists")
// 			return
// 		}
// 	}

// 	chat := NewChat(pair)
// 	Db.Chats = append(Db.Chats, chat)
// 	c.JSON(http.StatusOK, chat)
// }

// func createMessage(c *gin.Context) {
// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return
// 	}

// 	ix := slices.IndexFunc(Db.Chats, func(dbChat *Chat) bool {
// 		return dbChat.ID == id
// 	})

// 	if ix == -1 {
// 		slog.Error("trying to post message to nonexistent chat")
// 		return
// 	}

// 	var msg Message
// 	if err := c.BindJSON(&msg); err != nil {
// 		slog.Error("failed to bind message", "error", err)
// 		return
// 	}

// 	Db.Chats[ix].Messages = append(Db.Chats[ix].Messages, &msg)
// }
