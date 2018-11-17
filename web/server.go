package main

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vwxyzjn/portwarden/web/controllers"
)

func main() {
	r := gin.Default()
	r.Use(cors.Default())

	r.GET("/", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "index.html")
	})

	r.POST("/encrypt", controllers.EncryptBackupController)
	r.POST("/decrypt", controllers.DecryptBackupController)

	r.Run(":5000")
}
