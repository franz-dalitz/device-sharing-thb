package main

import (
	"github.com/franz-dalitz/device-sharing-thb/internal"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080" // fallback for local development
    }
	engine := internal.Server()
	internal.Db.Mock()
	engine.Run(":" + port)
}
