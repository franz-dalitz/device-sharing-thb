package data

var midg = IDGen{}

type Message struct {
	ID      int
	By      int
	Content string
}

func NewMessage(by int, content string) *Message {
	return &Message{
		midg.Generate(),
		by,
		content,
	}
}
