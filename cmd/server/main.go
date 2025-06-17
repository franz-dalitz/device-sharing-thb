package main

import (
	"github.com/franz-dalitz/device-sharing-thb/internal"
	"log"
	"os"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080" // fallback for local development
    }
    log.Println("Listening on port:", port)
	engine := internal.Server()
	internal.Db.Mock()
	engine.Run(":" + port)
}
