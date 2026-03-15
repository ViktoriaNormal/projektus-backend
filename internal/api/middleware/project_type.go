package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

// RequireProjectType проверяет, что проект имеет ожидаемый тип (scrum/kanban).
func RequireProjectType(requiredType domain.ProjectType, projectService *services.ProjectService) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectIDStr := c.Param("projectId")
		if projectIDStr == "" {
			writeTypeError(c, "VALIDATION_ERROR", "Не указан идентификатор проекта")
			return
		}
		projectID, err := uuid.Parse(projectIDStr)
		if err != nil {
			writeTypeError(c, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
			return
		}

		project, err := projectService.GetProject(c.Request.Context(), projectID)
		if err != nil {
			writeTypeError(c, "NOT_FOUND", "Проект не найден")
			return
		}

		if project.Type != requiredType {
			writeTypeError(c, "INVALID_PROJECT_TYPE", "Тип проекта не поддерживает запрошенную аналитику")
			return
		}

		c.Next()
	}
}

func writeTypeError(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, dto.APIResponse{
		Success: false,
		Data:    nil,
		Error: &dto.APIError{
			Code:    code,
			Message: message,
		},
	})
}

