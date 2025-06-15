package main

import (
	"github.com/franz-dalitz/device-sharing-thb/internal"
)

func main() {
	engine := internal.Server()
	internal.Db.Mock()
	engine.Run(":8080")
}
