package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/services"
)

func RequireSystemPermission(permission string, permissionService *services.PermissionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetString("userID")
		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, dto.APIResponse{
				Success: false,
				Data:    nil,
				Error: &dto.APIError{
					Code:    "UNAUTHORIZED",
					Message: "Требуется аутентификация",
				},
			})
			c.Abort()
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, dto.APIResponse{
				Success: false,
				Data:    nil,
				Error: &dto.APIError{
					Code:    "UNAUTHORIZED",
					Message: "Неверный идентификатор пользователя",
				},
			})
			c.Abort()
			return
		}

		has, err := permissionService.HasPermission(c.Request.Context(), userID, permission, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.APIResponse{
				Success: false,
				Data:    nil,
				Error: &dto.APIError{
					Code:    "INTERNAL_ERROR",
					Message: "Ошибка проверки прав доступа",
				},
			})
			c.Abort()
			return
		}

		if !has {
			c.JSON(http.StatusForbidden, dto.APIResponse{
				Success: false,
				Data:    nil,
				Error: &dto.APIError{
					Code:    "FORBIDDEN",
					Message: "Недостаточно прав для выполнения операции",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

