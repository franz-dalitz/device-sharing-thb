package data

type Database struct {
	Users   []*User
	Devices []*Device
	Chats   []*Chat
}

func (db *Database) Mock() {
	db.Users = append(db.Users,
		NewUser("Annalena Schmidt", "aschmidt@th-brandenburg.de"),
		NewUser("Karl Weber", "kweber@th-brandenburg.de"),
		NewUser("Willy Brechten", "wbrechten@th-brandenburg.de"),
	)

	db.Devices = append(db.Devices,
		NewDevice(db.Users[0].ID, "iPhone 14 Pro", Smartphone, "Personal smartphone, primary communication device.", "Office Desk").WithPhoto("iphone14pro.jpg"),
		NewDevice(db.Users[0].ID, "MacBook Pro 2023", Laptop, "Work laptop, used for coding and design.", "Office Desk").WithPhoto("macbookpro23.jpg"),
		NewDevice(db.Users[0].ID, "USB-C to Lightning Cable", Cable, "Charging cable for iPhone.", "Desk Drawer").WithPhoto("usblightning.jpg"),
		NewDevice(db.Users[1].ID, "Samsung Galaxy S23", Smartphone, "Personal Android phone.", "Living Room Table").WithPhoto("samsunggalaxys23.jpg"),
		NewDevice(db.Users[1].ID, "Anker Power Bank", Charger, "Portable charger for phones and tablets.", "Backpack").WithPhoto("ankerpowerbank.png"),
		NewDevice(db.Users[1].ID, "iPad Air 5th Gen", Tablet, "Used for reading and casual browsing.", "Bedroom Nightstand").WithPhoto("ipad5thgen.jpg"),
		NewDevice(db.Users[1].ID, "HP Spectre x360", Laptop, "Personal laptop for entertainment and light work.", "Living Room Table").WithPhoto("hpspectre.jpg"),
		NewDevice(db.Users[2].ID, "Google Pixel 7", Smartphone, "Secondary work phone.", "Car Console").WithPhoto("googlepixel7.jpg"),
		NewDevice(db.Users[2].ID, "Wireless Charging Pad", Charger, "Desk charger for multiple devices.", "Office Desk").WithPhoto("wirelesscharger.jpg"),
		NewDevice(db.Users[2].ID, "External SSD 1TB", Other, "Portable storage for backups.", "Home Office Shelf").WithPhoto("externalssd.jpg"),
	)

	db.Chats = append(db.Chats,
		NewChat(IntPair{db.Users[0].ID, db.Users[1].ID}),
	)

	db.Chats[0].Messages = append(db.Chats[0].Messages,
		NewMessage(db.Chats[0].Participants.X, "Hey!"),
		NewMessage(db.Chats[0].Participants.Y, "Selber hey..."),
	)
}
