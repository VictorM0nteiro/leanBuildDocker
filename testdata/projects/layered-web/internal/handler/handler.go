package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"example.com/layered-web/internal/service"
)

// Register wires HTTP routes — a middle layer between main and the service.
func Register(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": service.Status()})
	})
}
