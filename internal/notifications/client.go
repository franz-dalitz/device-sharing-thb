package notifications

import (
	"bytes"
	"html/template"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	UserID int
	Events chan *Event
}

func NewClient(h *Hub, c *websocket.Conn, u int) *Client {
	return &Client{
		Hub:    h,
		Conn:   c,
		UserID: u,
		Events: make(chan *Event, 10),
	}
}

func (c *Client) EventPump() {
	defer c.Conn.Close()
	for {
		event, ok := <-c.Events
		if !ok {
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		w, err := c.Conn.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}

		toast, err := template.ParseFiles("web/components/toast.tmpl")
		if err != nil {
			slog.Error(err.Error(), "err", err)
			return
		}

		var msg bytes.Buffer
		toast.Execute(&msg, event.Content)

		w.Write(msg.Bytes())
		if err := w.Close(); err != nil {
			return
		}

		time.Sleep(5 * time.Second)
	}
}
