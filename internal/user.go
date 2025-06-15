package internal

var uidg = IDGen{}

type User struct {
	ID    int
	Name  string
	Mail  string
	Liked []int
}

func NewUser(name string, mail string) *User {
	return &User{
		uidg.Generate(),
		name,
		mail,
		[]int{},
	}
}
