package internal

type Category int

const (
	Smartphone Category = iota + 1
	Charger
	Laptop
	Tablet
	Cable
	Other
)

var Categories = []string{
	"Smartphone",
	"Charger",
	"Laptop",
	"Tablet",
	"Cable",
	"Other",
}

func (cat Category) String() string {
	if cat < 0 || int(cat) > len(Categories) {
		return "Unknown"
	}
	return Categories[cat]
}
