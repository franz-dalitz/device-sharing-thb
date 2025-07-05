package data

import "time"

var uidg = IDGen{}

type User struct {
	ID    int
	Name  string
	Mail  string
	Liked []int
	Seen  time.Time
}

func NewUser(name string, mail string) *User {
	return &User{
		ID:   uidg.Generate(),
		Name: name,
		Mail: mail,
		Seen: time.Now().Add(-24 * 7 * time.Hour),
	}
}
