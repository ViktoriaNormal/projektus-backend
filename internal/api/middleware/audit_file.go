package middleware

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditFileLogger returns a middleware that logs every request to a file.
// Each line: timestamp | user_id (or "anonymous") | method | path | status | latency
func AuditFileLogger(filePath string) gin.HandlerFunc {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open audit log file %s: %v", filePath, err)
	}
	logger := log.New(f, "", 0)

	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			path = path + "?" + c.Request.URL.RawQuery
		}
		clientIP := c.ClientIP()

		userID := c.GetString("userID")
		if userID == "" {
			userID = "anonymous"
		}

		logger.Printf("%s | %s | %s | %s | %d | %s | %s",
			time.Now().UTC().Format(time.RFC3339),
			userID,
			method,
			path,
			status,
			latency.Truncate(time.Millisecond),
			clientIP,
		)
	}
}
