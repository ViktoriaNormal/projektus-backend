package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"projektus-backend/config"
	"projektus-backend/internal/api/dto"
	"projektus-backend/pkg/utils"
)

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			writeUnauthorized(c, "UNAUTHORIZED", "Требуется токен доступа")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeUnauthorized(c, "INVALID_AUTH_HEADER", "Некорректный заголовок авторизации")
			return
		}

		claims, err := utils.ParseAccessToken(cfg.JWTAccessSecret, parts[1])
		if err != nil {
			writeUnauthorized(c, "INVALID_TOKEN", "Недействительный или истекший токен")
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Next()
	}
}

func writeUnauthorized(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, dto.APIResponse{
		Success: false,
		Data:    nil,
		Error: &dto.APIError{
			Code:    code,
			Message: message,
		},
	})
}

