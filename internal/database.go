package internal

var Db Database = Database{}

type Database struct {
	Users   []*User
	Chats   []*Chat
	Devices []*Device
}

func (db *Database) Mock() {
	db.Users = append(db.Users,
		NewUser("Annalena Schmidt", "aschmidt@th-brandenburg.de"),
		NewUser("Karl Weber", "kweber@th-brandenburg.de"),
		NewUser("Willy Brechten", "wbrechten@th-brandenburg.de"),
	)

	db.Devices = append(db.Devices,
		NewDevice(db.Users[0].ID, "Altes Smartphone", Smartphone, "", ""),
		NewDevice(db.Users[1].ID, "Neuer Laptop", Laptop, "", ""),
		NewDevice(db.Users[2].ID, "Altes Kabel", Cable, "", ""),
	)
}
