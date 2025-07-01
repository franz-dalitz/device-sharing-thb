package notifications

type Hub struct {
	Clients    map[*Client]bool
	Register   chan *Client
	Unregister chan *Client
	Deliver    chan *Event
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Deliver:    make(chan *Event, 10),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Events)
			}
		case event := <-h.Deliver:
			for client := range h.Clients {
				if event.Recipient != client.UserID {
					continue
				}
				select {
				case client.Events <- event:
				default:
					close(client.Events)
					delete(h.Clients, client)
				}
			}
		}
	}
}
