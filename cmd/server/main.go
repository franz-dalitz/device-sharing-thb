package main

import (
	"log"
	"os"

	"github.com/franz-dalitz/device-sharing-thb/internal"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // fallback for local development
	}
	log.Println("Listening on port:", port)
	engine := internal.Server()
	engine.Run(":" + port)
}
