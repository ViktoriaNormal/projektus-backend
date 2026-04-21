package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"projektus-backend/internal/api/dto"
)

// writeError пишет единый формат ошибочного ответа:
//
//	{ "success": false, "data": null, "error": { "code": ..., "message": ... } }
//
// Код передаётся в стандартном формате SCREAMING_SNAKE (например, "VALIDATION_ERROR",
// "NOT_FOUND") — фронтенд маршрутизирует именно по `code`.
func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, dto.APIResponse{
		Success: false,
		Data:    nil,
		Error: &dto.APIError{
			Code:    code,
			Message: message,
		},
	})
}

// writeSuccess пишет успешный ответ со стандартным envelope `{ success, data, error }`
// и статусом 200. Для 201-созданного ресурса используйте c.JSON(201, ...) напрямую.
func writeSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data:    data,
		Error:   nil,
	})
}
