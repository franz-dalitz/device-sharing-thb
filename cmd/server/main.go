package main

import (
	"log"
	"os"

	"github.com/franz-dalitz/device-sharing-thb/internal"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/franz-dalitz/device-sharing-thb/docs"
)

// @Title Swagger Example API
// @Version 1.0
// @License.Name Apache 2.0
func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // fallback for local development
	}
	log.Println("Listening on port:", port)
	engine := internal.Server()
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	engine.Run(":" + port)
}
