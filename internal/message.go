package internal

import "time"

var midg = IDGen{}

type Message struct {
	ID      int
	By      int
	Content string
	Time    time.Time
}

func NewMessage(by int, content string) *Message {
	return &Message{
		midg.Generate(),
		by,
		content,
		time.Now(),
	}
}
