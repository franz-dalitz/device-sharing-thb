package internal

import "image"

var didg = IDGen{}

type Device struct {
	ID          int
	Owner       int
	Title       string
	Category    Category
	Description string
	Location    string
	Photo       image.Image `json:"-"`
	ReservedBy  int
}

func NewDevice(owner int, name string, cat Category, desc string, loc string) *Device {
	return &Device{
		didg.Generate(),
		owner,
		name,
		cat,
		desc,
		loc,
		image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{0, 0}}),
		-1,
	}
}
