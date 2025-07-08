package notifications

type EventType int

const (
	Notification EventType = iota
	Message
)

type Event struct {
	Type      EventType
	Recipient int
	Content   string
}
