package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"projektus-backend/config"
)

func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", cfg.CORSAllowedOrigin)
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Max-Age", "86400")
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}
