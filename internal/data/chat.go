package data

import "errors"

var cidg = IDGen{}

type IntPair struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func (pair IntPair) Contains(val int) bool {
	return pair.X == val || pair.Y == val
}

func (pair IntPair) Other(val int) (int, error) {
	if val == pair.X {
		return pair.Y, nil
	} else if val == pair.Y {
		return pair.X, nil
	} else {
		return 0, errors.New("int not in pair")
	}
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
