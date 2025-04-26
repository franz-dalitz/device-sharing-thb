package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	enGin := gin.Default()

	enGin.LoadHTMLGlob("web/templates/*")
	enGin.Static("/static/htmx", "web/node_modules/htmx.org/dist")
	enGin.Static("/static/bootstrap", "web/node_modules/bootstrap/dist")

	enGin.GET("/", func(context *gin.Context) {
		context.HTML(http.StatusOK, "home.tmpl", gin.H{
			"dynamic": "this text was templated by the backend",
		})
	})

	enGin.GET("/test", func(context *gin.Context) {
		context.String(http.StatusOK, "funktioniert :)")
	})

	enGin.Run(":8080")
}
