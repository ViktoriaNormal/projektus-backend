package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type TaskHandler struct {
	service *services.TaskService
}

func NewTaskHandler(service *services.TaskService) *TaskHandler {
	return &TaskHandler{service: service}
}

func (h *TaskHandler) SearchTasks(c *gin.Context) {
	var req dto.SearchTasksRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные параметры фильтрации")
		return
	}
	tasks, err := h.service.SearchTasks(c.Request.Context(), req.ProjectID, req.OwnerID, req.ExecutorID, req.ColumnID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить список задач")
		return
	}
	resp := make([]dto.TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, mapTaskToDTO(&t))
	}
	writeSuccess(c, resp)
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	var deadline *time.Time
	if req.Deadline != nil {
		d := *req.Deadline
		deadline = &d
	}

	task, err := h.service.CreateTask(c.Request.Context(), req.ProjectID, req.OwnerMemberID, req.Name, req.Description, req.ExecutorMemberID, req.ColumnID, req.SwimlaneID, deadline)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные параметры задачи")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать задачу")
		return
	}
	writeSuccess(c, mapTaskToDTO(task))
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	idStr := c.Param("taskId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	task, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Задача не найдена")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить задачу")
		return
	}
	writeSuccess(c, mapTaskToDTO(task))
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
	idStr := c.Param("taskId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	var req dto.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	task, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Задача не найдена")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить задачу")
		return
	}

	if req.Name != nil {
		task.Name = *req.Name
	}
	if req.Description != nil {
		task.Description = req.Description
	}
	if req.Deadline != nil {
		d := *req.Deadline
		task.Deadline = &d
	}
	if req.ExecutorMemberID != nil {
		idStr := req.ExecutorMemberID.String()
		task.ExecutorID = &idStr
	}
	if req.ColumnID != nil {
		task.ColumnID = req.ColumnID.String()
	}
	if req.SwimlaneID != nil {
		idStr := req.SwimlaneID.String()
		task.SwimlaneID = &idStr
	}

	updated, err := h.service.UpdateTask(c.Request.Context(), task)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные параметры задачи")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить задачу")
		return
	}
	writeSuccess(c, mapTaskToDTO(updated))
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
	idStr := c.Param("taskId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	var req dto.DeleteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	if err := h.service.DeleteTask(c.Request.Context(), id, req.Reason); err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Для удаления задачи требуется указать причину")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить задачу")
		return
	}
	writeSuccess(c, gin.H{"message": "Задача удалена"})
}

func (h *TaskHandler) ListWatchers(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	watchers, err := h.service.ListWatchers(c.Request.Context(), taskID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить наблюдателей")
		return
	}
	resp := make([]dto.TaskWatcherResponse, 0, len(watchers))
	for _, w := range watchers {
		resp = append(resp, dto.TaskWatcherResponse{
			ID:              uuid.MustParse(w.ID),
			TaskID:          uuid.MustParse(w.TaskID),
			ProjectMemberID: uuid.MustParse(w.ProjectMemberID),
		})
	}
	writeSuccess(c, resp)
}

func (h *TaskHandler) AddWatcher(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	var req dto.AddWatcherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	watcher, err := h.service.AddWatcher(c.Request.Context(), taskID, req.ProjectMemberID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось добавить наблюдателя")
		return
	}
	writeSuccess(c, dto.TaskWatcherResponse{
		ID:              uuid.MustParse(watcher.ID),
		TaskID:          uuid.MustParse(watcher.TaskID),
		ProjectMemberID: uuid.MustParse(watcher.ProjectMemberID),
	})
}

func (h *TaskHandler) ListDependencies(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	deps, err := h.service.ListDependencies(c.Request.Context(), taskID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить зависимости")
		return
	}
	resp := make([]dto.TaskDependencyResponse, 0, len(deps))
	for _, d := range deps {
		resp = append(resp, dto.TaskDependencyResponse{
			ID:              uuid.MustParse(d.ID),
			TaskID:          uuid.MustParse(d.TaskID),
			DependsOnTaskID: uuid.MustParse(d.DependsOnTaskID),
			Type:            string(d.Type),
		})
	}
	writeSuccess(c, resp)
}

func (h *TaskHandler) AddDependency(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	var req dto.AddDependencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	depType := domain.TaskDependencyType(req.Type)
	dep, err := h.service.AddDependency(c.Request.Context(), taskID, req.DependsOnTaskID, depType)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный тип зависимости или самоссылка")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать зависимость")
		return
	}
	writeSuccess(c, dto.TaskDependencyResponse{
		ID:              uuid.MustParse(dep.ID),
		TaskID:          uuid.MustParse(dep.TaskID),
		DependsOnTaskID: uuid.MustParse(dep.DependsOnTaskID),
		Type:            string(dep.Type),
	})
}

func (h *TaskHandler) ListChecklists(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	checklists, err := h.service.ListChecklistsWithItems(c.Request.Context(), taskID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить чек-листы")
		return
	}
	resp := make([]dto.ChecklistResponse, 0, len(checklists))
	for _, ch := range checklists {
		resp = append(resp, mapChecklistToDTO(&ch))
	}
	writeSuccess(c, resp)
}

func (h *TaskHandler) CreateChecklist(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	var req dto.CreateChecklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	ch, err := h.service.CreateChecklist(c.Request.Context(), taskID, req.Name)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректное название чек-листа")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать чек-лист")
		return
	}
	writeSuccess(c, mapChecklistToDTO(ch))
}

func (h *TaskHandler) AddChecklistItem(c *gin.Context) {
	checklistIDStr := c.Param("checklistId")
	checklistID, err := uuid.Parse(checklistIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор чек-листа")
		return
	}
	var req dto.CreateChecklistItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	item, err := h.service.AddChecklistItem(c.Request.Context(), checklistID, req.Content, req.Order)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректное содержимое пункта чек-листа")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось добавить пункт чек-листа")
		return
	}
	writeSuccess(c, mapChecklistItemToDTO(item))
}

func (h *TaskHandler) SetChecklistItemStatus(c *gin.Context) {
	itemIDStr := c.Param("itemId")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор пункта чек-листа")
		return
	}
	var req dto.SetChecklistItemStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	item, err := h.service.SetChecklistItemStatus(c.Request.Context(), itemID, req.IsChecked)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить статус пункта чек-листа")
		return
	}
	writeSuccess(c, mapChecklistItemToDTO(item))
}

func mapChecklistToDTO(ch *domain.Checklist) dto.ChecklistResponse {
	taskID := uuid.MustParse(ch.TaskID)
	items := make([]dto.ChecklistItemResponse, 0, len(ch.Items))
	for _, it := range ch.Items {
		items = append(items, mapChecklistItemToDTO(&it))
	}
	return dto.ChecklistResponse{
		ID:     uuid.MustParse(ch.ID),
		TaskID: taskID,
		Name:   ch.Name,
		Items:  items,
	}
}

func mapChecklistItemToDTO(it *domain.ChecklistItem) dto.ChecklistItemResponse {
	return dto.ChecklistItemResponse{
		ID:          uuid.MustParse(it.ID),
		ChecklistID: uuid.MustParse(it.ChecklistID),
		Content:     it.Content,
		IsChecked:   it.IsChecked,
		Order:       it.Order,
	}
}


func mapTaskToDTO(t *domain.Task) dto.TaskResponse {
	projectID := uuid.MustParse(t.ProjectID)
	ownerID := uuid.MustParse(t.OwnerID)
	var execID *uuid.UUID
	if t.ExecutorID != nil {
		id := uuid.MustParse(*t.ExecutorID)
		execID = &id
	}
	var desc *string
	if t.Description != nil {
		desc = t.Description
	}
	var swimlaneID *uuid.UUID
	if t.SwimlaneID != nil {
		id := uuid.MustParse(*t.SwimlaneID)
		swimlaneID = &id
	}
	var deadline *time.Time
	if t.Deadline != nil {
		d := *t.Deadline
		deadline = &d
	}

	var progress *int
	if len(t.Checklists) > 0 {
		total := 0
		done := 0
		for _, ch := range t.Checklists {
			for _, it := range ch.Items {
				total++
				if it.IsChecked {
					done++
				}
			}
		}
		if total > 0 {
			p := int(float64(done) / float64(total) * 100.0)
			progress = &p
		}
	}

	return dto.TaskResponse{
		ID:         uuid.MustParse(t.ID),
		Key:        t.Key,
		ProjectID:  projectID,
		OwnerID:    ownerID,
		ExecutorID: execID,
		Name:       t.Name,
		Description: desc,
		Deadline:   deadline,
		ColumnID:   uuid.MustParse(t.ColumnID),
		SwimlaneID: swimlaneID,
		Progress:   progress,
	}
}

