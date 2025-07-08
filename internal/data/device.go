package data

var didg = IDGen{}

type Device struct {
	ID          int
	Owner       int
	Title       string
	Category    Category
	Description string
	Location    string
	Photo       string
	ReservedBy  int
}

func NewDevice(owner int, title string, cat Category, desc string, loc string) *Device {
	return &Device{
		ID:          didg.Generate(),
		Owner:       owner,
		Title:       title,
		Category:    cat,
		Description: desc,
		Location:    loc,
		ReservedBy:  -1,
	}
}

func (device *Device) WithPhoto(photo string) *Device {
	device.Photo = photo
	return device
}
