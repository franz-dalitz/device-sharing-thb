package notifications

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"text/template"
	"time"

	"github.com/franz-dalitz/device-sharing-thb/internal/data"
	"github.com/gorilla/websocket"
)

const (
	pongWait     = 60 * time.Second
	writeWait    = 10 * time.Second
	pingInterval = (pongWait * 9) / 10
	seenInterval = 10 * time.Second
)

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	user   *data.User
	events chan *Event
	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once
}

func NewClient(h *Hub, c *websocket.Conn, u *data.User) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		hub:    h,
		conn:   c,
		user:   u,
		events: make(chan *Event, 10),
		ctx:    ctx,
		cancel: cancel,
		once:   sync.Once{},
	}
}

func (c *Client) seen() {
	ticker := time.NewTicker(seenInterval)
	defer ticker.Stop()
	defer c.exit()

	c.user.Seen = time.Now()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.user.Seen = time.Now()
		}
	}
}

func (c *Client) exit() {
	c.once.Do(func() {
		c.hub.Unregister <- c
		c.conn.Close()
		c.cancel()
		close(c.events)
	})
}

func (c *Client) read() {
	defer c.exit()
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) write() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	defer c.exit()

	toastTmpl, err := template.ParseFiles("web/components/toast.tmpl")
	if err != nil {
		slog.Error(err.Error())
		return
	}

	messagesTmpl, err := template.ParseFiles("web/components/other-message.tmpl")
	if err != nil {
		slog.Error(err.Error())
		return
	}

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case evt, ok := <-c.events:
			if !ok {
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				slog.Error(err.Error())
				return
			}
			var msg bytes.Buffer
			if evt.Type == Notification {
				if err := toastTmpl.Execute(&msg, evt.Content); err != nil {
					slog.Error(err.Error())
					continue
				}
			} else if evt.Type == Message {
				if err := messagesTmpl.Execute(&msg, evt.Content); err != nil {
					slog.Error(err.Error())
					continue
				}
			}
			w.Write(msg.Bytes())
			if err := w.Close(); err != nil {
				slog.Error(err.Error())
				return
			}
		}
	}
}

func (c *Client) Run() {
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	go c.read()
	go c.write()
	go c.seen()
}
