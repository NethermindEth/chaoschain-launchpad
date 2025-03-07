package api

import (
	"github.com/gin-gonic/gin"
)

// StartServer initializes the REST API
func StartServer() {
	r := gin.Default()

	r.Run(":8080") // Default port
}
