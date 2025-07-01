package notifications

import "github.com/gorilla/websocket"

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

		w.Write([]byte(event.Content))
		if err := w.Close(); err != nil {
			return
		}
	}
}
