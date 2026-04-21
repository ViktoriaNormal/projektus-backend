package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type SprintBacklogHandler struct {
	sprintService *services.SprintService
}

func NewSprintBacklogHandler(sprintService *services.SprintService) *SprintBacklogHandler {
	return &SprintBacklogHandler{sprintService: sprintService}
}

func (h *SprintBacklogHandler) GetSprintBacklog(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}
	sprint, err := h.sprintService.GetActiveSprint(c.Request.Context(), projectID)
	if err != nil {
		// Нет активного спринта — пустой бэклог (исторический контракт, не 404).
		if err == domain.ErrNotFound {
			writeSuccess(c, []dto.TaskResponse{})
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить активный спринт")
		return
	}
	tasks, err := h.sprintService.GetSprintBacklog(c.Request.Context(), sprint.ID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить бэклог спринта")
		return
	}
	resp := make([]dto.TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, mapTaskToDTO(&t))
	}
	writeSuccess(c, resp)
}

type moveToSprintRequest struct {
	SprintID uuid.UUID   `json:"sprint_id" binding:"required"`
	TaskIDs  []uuid.UUID `json:"task_ids" binding:"required,min=1"`
}

func (h *SprintBacklogHandler) MoveTasksToSprint(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}
	req, ok := bindJSON[moveToSprintRequest](c)
	if !ok {
		return
	}
	if err := h.sprintService.MoveTasksToSprint(c.Request.Context(), req.SprintID, projectID, req.TaskIDs); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось перенести задачи в спринт")
		return
	}
	writeSuccess(c, gin.H{"message": "Задачи перенесены в спринт"})
}
