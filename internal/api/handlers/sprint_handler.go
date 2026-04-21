package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type SprintHandler struct {
	service    *services.SprintService
	projectSvc *services.ProjectService
}

func NewSprintHandler(service *services.SprintService, projectSvc *services.ProjectService) *SprintHandler {
	return &SprintHandler{service: service, projectSvc: projectSvc}
}

func (h *SprintHandler) ListProjectSprints(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}
	sprints, err := h.service.GetProjectSprints(c.Request.Context(), projectID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить список спринтов")
		return
	}
	resp := make([]dto.SprintResponse, 0, len(sprints))
	for _, s := range sprints {
		resp = append(resp, mapSprintToDTO(&s))
	}
	writeSuccess(c, resp)
}

func (h *SprintHandler) CreateSprint(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.CreateSprintRequest](c)
	if !ok {
		return
	}

	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректная дата начала спринта")
		return
	}

	duration := 0
	if req.DurationDays != nil {
		duration = *req.DurationDays
	} else if req.DurationWeeks != nil {
		duration = *req.DurationWeeks * 7
	}
	if duration <= 0 {
		// Use project's sprint_duration_weeks as default, fallback to 2 weeks
		project, projErr := h.projectSvc.GetProject(c.Request.Context(), projectID)
		if projErr == nil && project.SprintDurationWeeks != nil {
			duration = *project.SprintDurationWeeks * 7
		} else {
			duration = 14
		}
	}

	sprint, err := h.service.CreateSprint(c.Request.Context(), projectID, req.Name, req.Goal, start, duration)
	if err != nil {
		// Сохраняем исторический маппинг — VALIDATION_ERROR вместо SPRINT_DATES_OVERLAP.
		if err == domain.ErrSprintDatesOverlap {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Даты спринта пересекаются с существующим незавершённым спринтом")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать спринт")
		return
	}
	writeSuccess(c, mapSprintToDTO(sprint))
}

func (h *SprintHandler) GetSprint(c *gin.Context) {
	id, ok := paramUUID(c, "sprintId")
	if !ok {
		return
	}
	sprint, err := h.service.GetSprint(c.Request.Context(), id)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить спринт")
		return
	}
	writeSuccess(c, mapSprintToDTO(sprint))
}

func (h *SprintHandler) UpdateSprint(c *gin.Context) {
	id, ok := paramUUID(c, "sprintId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.UpdateSprintRequest](c)
	if !ok {
		return
	}
	sprint, err := h.service.GetSprint(c.Request.Context(), id)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить спринт")
		return
	}

	// Apply nullable goal field in handler (three-state: absent/null/value).
	if req.Goal.Set {
		sprint.Goal = req.Goal.Ptr()
	}

	var startPtr *time.Time
	if req.StartDate != nil {
		t, err := time.Parse("2006-01-02", *req.StartDate)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректная дата начала спринта")
			return
		}
		startPtr = &t
	}

	var duration *int
	if req.DurationDays != nil {
		duration = req.DurationDays
	} else if req.DurationWeeks != nil {
		d := *req.DurationWeeks * 7
		duration = &d
	}

	updated, err := h.service.UpdateSprint(c.Request.Context(), sprint, req.Name, nil, startPtr, duration)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить спринт")
		return
	}
	writeSuccess(c, mapSprintToDTO(updated))
}

func (h *SprintHandler) DeleteSprint(c *gin.Context) {
	id, ok := paramUUID(c, "sprintId")
	if !ok {
		return
	}
	if err := h.service.DeleteSprint(c.Request.Context(), id); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить спринт")
		return
	}
	writeSuccess(c, gin.H{"message": "Спринт удалён"})
}

func (h *SprintHandler) StartSprint(c *gin.Context) {
	id, ok := paramUUID(c, "sprintId")
	if !ok {
		return
	}
	sprint, err := h.service.StartSprint(c.Request.Context(), id)
	if err != nil {
		// Сохраняем исторический маппинг — VALIDATION_ERROR вместо ACTIVE_SPRINT_EXISTS.
		if err == domain.ErrActiveSprintExists {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Нельзя запустить спринт: уже есть активный спринт на проекте")
			return
		}
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Нельзя запустить завершённый спринт")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось запустить спринт")
		return
	}
	writeSuccess(c, mapSprintToDTO(sprint))
}

func (h *SprintHandler) CompleteSprint(c *gin.Context) {
	id, ok := paramUUID(c, "sprintId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.CompleteSprintRequest](c)
	if !ok {
		return
	}
	sprint, err := h.service.CompleteSprint(c.Request.Context(), id, req.IncompleteTasksAction)
	if err != nil {
		// Сохраняем исторический маппинг — VALIDATION_ERROR вместо NO_NEXT_SPRINT.
		if err == domain.ErrNoNextSprintForMove {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Нет следующего спринта для переноса незавершённых задач")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось завершить спринт")
		return
	}
	writeSuccess(c, mapSprintToDTO(sprint))
}

func (h *SprintHandler) GetSprintTasks(c *gin.Context) {
	id, ok := paramUUID(c, "sprintId")
	if !ok {
		return
	}
	tasks, err := h.service.GetSprintTasks(c.Request.Context(), id)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить задачи спринта")
		return
	}
	resp := make([]dto.TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, mapTaskToDTO(&t))
	}
	writeSuccess(c, resp)
}

func mapSprintToDTO(s *domain.Sprint) dto.SprintResponse {
	return dto.SprintResponse{
		ID:        s.ID,
		ProjectID: s.ProjectID,
		Name:      s.Name,
		Goal:      s.Goal,
		StartDate: s.StartDate.Format("2006-01-02"),
		EndDate:   s.EndDate.Format("2006-01-02"),
		Status:    string(s.Status),
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
		UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
	}
}
