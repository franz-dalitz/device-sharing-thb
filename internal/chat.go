package internal

var cidg = IDGen{}

type IntPair struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func (pair IntPair) Contains(val int) bool {
	return pair.X == val || pair.Y == val
}

func (left IntPair) Equals(right IntPair) bool {
	return (left.X == right.X && left.Y == right.Y) || (left.X == right.Y && left.Y == right.X)
}

type Chat struct {
	ID           int
	Participants IntPair
	Messages     []*Message
}

func NewChat(ip IntPair) *Chat {
	return &Chat{
		cidg.Generate(),
		ip,
		[]*Message{},
	}
}
