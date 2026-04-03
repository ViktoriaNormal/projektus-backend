package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type TaskHandler struct {
	service    *services.TaskService
	boardSvc   *services.BoardService
	projectSvc *services.ProjectService
}

func NewTaskHandler(service *services.TaskService, boardSvc *services.BoardService, projectSvc *services.ProjectService) *TaskHandler {
	return &TaskHandler{service: service, boardSvc: boardSvc, projectSvc: projectSvc}
}

func (h *TaskHandler) SearchTasks(c *gin.Context) {
	var req dto.SearchTasksRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные параметры фильтрации")
		return
	}
	userIDStr := c.GetString("userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Не удалось определить пользователя")
		return
	}
	var projectID, columnID *uuid.UUID
	if req.ProjectID != nil {
		id, err := uuid.Parse(*req.ProjectID)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
			return
		}
		projectID = &id
	}
	if req.ColumnID != nil {
		id, err := uuid.Parse(*req.ColumnID)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор колонки")
			return
		}
		columnID = &id
	}
	tasks, err := h.service.SearchTasks(c.Request.Context(), userID, projectID, columnID)
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

	// Определяем тип проекта для корректной логики назначения column_id
	project, err := h.projectSvc.GetProject(c.Request.Context(), req.ProjectID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Не удалось получить проект")
		return
	}

	columnID := req.ColumnID
	isScrum := project.Type == domain.ProjectTypeScrum

	if isScrum && req.AddToBacklog {
		// Scrum + backlog: column_id остаётся NULL — назначится при запуске спринта
		columnID = uuid.Nil
	} else if req.BoardID != nil && columnID == uuid.Nil {
		// Kanban или Scrum без backlog: назначаем начальную колонку
		columns, err := h.boardSvc.ListColumns(c.Request.Context(), *req.BoardID)
		if err != nil {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Не удалось получить колонки доски")
			return
		}
		found := false
		for _, col := range columns {
			if col.SystemType != nil && *col.SystemType == domain.StatusInitial {
				columnID = uuid.MustParse(col.ID)
				found = true
				break
			}
		}
		if !found {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Доска не содержит начальной колонки")
			return
		}
	}

	if columnID == uuid.Nil && !isScrum {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Необходимо указать column_id или board_id")
		return
	}

	// Build params for full creation
	params := services.CreateTaskFullParams{
		ProjectID:        req.ProjectID,
		OwnerMemberID:    req.OwnerMemberID,
		Name:             req.Name,
		Description:      req.Description,
		ExecutorMemberID: req.ExecutorMemberID,
		ColumnID:         columnID,
		BoardID:          req.BoardID,
		SwimlaneID:       req.SwimlaneID,
		Deadline:         req.Deadline,
		Priority:         req.Priority,
		Estimation:       req.Estimation,
		WatcherMemberIDs: req.WatcherMemberIDs,
		Tags:             req.Tags,
		AddToBacklog:     req.AddToBacklog,
	}

	for _, cl := range req.Checklists {
		cp := services.CreateChecklistParam{Name: cl.Name}
		for _, item := range cl.Items {
			cp.Items = append(cp.Items, services.CreateChecklistItemParam{
				Content:   item.Content,
				IsChecked: item.IsChecked,
				Order:     int16(item.Order),
			})
		}
		params.Checklists = append(params.Checklists, cp)
	}

	for _, fv := range req.FieldValues {
		params.FieldValues = append(params.FieldValues, services.CreateFieldValueParam{
			FieldID:       fv.FieldID,
			ValueText:     fv.ValueText,
			ValueNumber:   fv.ValueNumber,
			ValueDatetime: fv.ValueDatetime,
		})
	}

	for _, dep := range req.Dependencies {
		params.Dependencies = append(params.Dependencies, services.CreateDependencyParam{
			DependsOnTaskID: dep.DependsOnTaskID,
			Type:            domain.TaskDependencyType(dep.Type),
		})
	}

	createdTask, err := h.service.CreateTaskFull(c.Request.Context(), params)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные параметры задачи")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать задачу: "+err.Error())
		return
	}
	// Re-fetch to get full data with user_id JOINs
	task, err := h.service.GetTask(c.Request.Context(), uuid.MustParse(createdTask.ID))
	if err != nil {
		// Fallback: return created task without user_ids
		writeSuccess(c, mapTaskToDTO(createdTask))
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
	if req.Description.Set {
		task.Description = req.Description.Ptr()
	}
	if req.Deadline.Set {
		task.Deadline = req.Deadline.Ptr()
	}
	if req.ExecutorMemberID.Set {
		if req.ExecutorMemberID.Null {
			task.ExecutorID = nil
		} else {
			idStr := req.ExecutorMemberID.Value.String()
			task.ExecutorID = &idStr
		}
	}
	if req.ColumnID != nil {
		s := req.ColumnID.String()
		task.ColumnID = &s
	}
	if req.SwimlaneID.Set {
		if req.SwimlaneID.Null {
			task.SwimlaneID = nil
		} else {
			idStr := req.SwimlaneID.Value.String()
			task.SwimlaneID = &idStr
		}
	}
	if req.Priority.Set {
		if req.Priority.Null {
			task.Priority = nil
		} else {
			task.Priority = &req.Priority.Value
		}
	}
	if req.Estimation.Set {
		if req.Estimation.Null {
			task.Estimation = nil
		} else {
			task.Estimation = &req.Estimation.Value
		}
	}

	updated, err := h.service.UpdateTask(c.Request.Context(), task, c.GetString("userID"))
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
	if err := h.service.DeleteTask(c.Request.Context(), id); err != nil {
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
			TaskID:   uuid.MustParse(w.TaskID),
			MemberID: uuid.MustParse(w.MemberID),
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
	if err := h.service.AddWatcher(c.Request.Context(), taskID, req.MemberID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось добавить наблюдателя")
		return
	}
	writeSuccess(c, dto.TaskWatcherResponse{
		TaskID:   taskID,
		MemberID: req.MemberID,
	})
}

func (h *TaskHandler) RemoveWatcher(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	memberIDStr := c.Param("memberId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор участника")
		return
	}
	if err := h.service.RemoveWatcher(c.Request.Context(), taskID, memberID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить наблюдателя")
		return
	}
	writeSuccess(c, gin.H{"message": "Наблюдатель удалён"})
}

// Comments

func (h *TaskHandler) ListComments(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	comments, err := h.service.ListComments(c.Request.Context(), taskID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить комментарии")
		return
	}
	resp := make([]dto.CommentResponse, 0, len(comments))
	for _, cm := range comments {
		resp = append(resp, mapCommentToDTO(&cm))
	}
	writeSuccess(c, resp)
}

func (h *TaskHandler) CreateComment(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	userIDStr := c.GetString("userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Не удалось определить пользователя")
		return
	}
	comment, err := h.service.CreateComment(c.Request.Context(), taskID, userID, req.Content, req.ParentCommentID)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Содержимое комментария не может быть пустым")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать комментарий")
		return
	}
	writeSuccess(c, mapCommentToDTO(comment))
}

func (h *TaskHandler) DeleteComment(c *gin.Context) {
	commentIDStr := c.Param("commentId")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор комментария")
		return
	}
	if err := h.service.DeleteComment(c.Request.Context(), commentID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить комментарий")
		return
	}
	writeSuccess(c, gin.H{"message": "Комментарий удалён"})
}

// Attachments

func (h *TaskHandler) ListAttachments(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	attachments, err := h.service.ListAttachments(c.Request.Context(), taskID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить вложения")
		return
	}
	resp := make([]dto.AttachmentResponse, 0, len(attachments))
	for _, a := range attachments {
		resp = append(resp, mapAttachmentToDTO(&a))
	}
	writeSuccess(c, resp)
}

func (h *TaskHandler) UploadAttachment(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	userIDStr := c.GetString("userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Не удалось определить пользователя")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Файл не найден в запросе")
		return
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	uniqueName := uuid.New().String() + ext
	dir := "uploads/attachments"
	if err := os.MkdirAll(dir, 0755); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось создать директорию для файлов")
		return
	}
	filePath := dir + "/" + uniqueName

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось сохранить файл")
		return
	}

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	attachment, err := h.service.CreateAttachment(c.Request.Context(), taskID, userID, file.Filename, filePath, contentType, file.Size)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось сохранить информацию о вложении")
		return
	}
	writeSuccess(c, mapAttachmentToDTO(attachment))
}

func (h *TaskHandler) DownloadAttachment(c *gin.Context) {
	attachmentIDStr := c.Param("attachmentId")
	attachmentID, err := uuid.Parse(attachmentIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор вложения")
		return
	}
	attachment, err := h.service.GetAttachmentByID(c.Request.Context(), attachmentID)
	if err != nil {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "Вложение не найдено")
		return
	}
	c.FileAttachment(attachment.FilePath, attachment.FileName)
}

func (h *TaskHandler) DeleteAttachment(c *gin.Context) {
	attachmentIDStr := c.Param("attachmentId")
	attachmentID, err := uuid.Parse(attachmentIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор вложения")
		return
	}
	if err := h.service.DeleteAttachment(c.Request.Context(), attachmentID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить вложение")
		return
	}
	writeSuccess(c, gin.H{"message": "Вложение удалено"})
}

// Field values

func (h *TaskHandler) GetTaskFieldValues(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	fieldValues, err := h.service.GetTaskFieldValues(c.Request.Context(), taskID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить значения полей")
		return
	}
	resp := make([]dto.TaskFieldValueResponse, 0, len(fieldValues))
	for _, fv := range fieldValues {
		resp = append(resp, dto.TaskFieldValueResponse{
			FieldID:       uuid.MustParse(fv.FieldID),
			ValueText:     fv.ValueText,
			ValueNumber:   fv.ValueNumber,
			ValueDatetime: fv.ValueDatetime,
		})
	}
	writeSuccess(c, resp)
}

func (h *TaskHandler) SetTaskFieldValue(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор задачи")
		return
	}
	fieldIDStr := c.Param("fieldId")
	fieldID, err := uuid.Parse(fieldIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор поля")
		return
	}
	var req dto.SetTaskFieldValueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	if err := h.service.UpsertTaskFieldValue(c.Request.Context(), taskID, fieldID, req.ValueText, req.ValueNumber, req.ValueDatetime); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось сохранить значение поля")
		return
	}
	writeSuccess(c, gin.H{"message": "Значение поля сохранено"})
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
		if err == domain.ErrConflict {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Связь между этими задачами уже существует")
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
	item, err := h.service.AddChecklistItem(c.Request.Context(), checklistID, req.Content, int16(req.Order))
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
	item, err := h.service.SetChecklistItemStatus(c.Request.Context(), itemID, *req.IsChecked)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить статус пункта чек-листа")
		return
	}
	writeSuccess(c, mapChecklistItemToDTO(item))
}

func (h *TaskHandler) UpdateChecklist(c *gin.Context) {
	checklistIDStr := c.Param("checklistId")
	checklistID, err := uuid.Parse(checklistIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор чек-листа")
		return
	}
	var req dto.UpdateChecklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	ch, err := h.service.UpdateChecklistName(c.Request.Context(), checklistID, req.Name)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Название чек-листа не может быть пустым")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить чек-лист")
		return
	}
	writeSuccess(c, mapChecklistToDTO(ch))
}

func (h *TaskHandler) DeleteChecklist(c *gin.Context) {
	checklistIDStr := c.Param("checklistId")
	checklistID, err := uuid.Parse(checklistIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор чек-листа")
		return
	}
	if err := h.service.DeleteChecklist(c.Request.Context(), checklistID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить чек-лист")
		return
	}
	writeSuccess(c, gin.H{"message": "Чек-лист удалён"})
}

func (h *TaskHandler) UpdateChecklistItem(c *gin.Context) {
	itemIDStr := c.Param("itemId")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор пункта чек-листа")
		return
	}
	var req dto.UpdateChecklistItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}
	item, err := h.service.UpdateChecklistItemContent(c.Request.Context(), itemID, req.Content)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Содержимое пункта не может быть пустым")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить пункт чек-листа")
		return
	}
	writeSuccess(c, mapChecklistItemToDTO(item))
}

func (h *TaskHandler) DeleteChecklistItem(c *gin.Context) {
	itemIDStr := c.Param("itemId")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор пункта чек-листа")
		return
	}
	if err := h.service.DeleteChecklistItem(c.Request.Context(), itemID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось удалить пункт чек-листа")
		return
	}
	writeSuccess(c, gin.H{"message": "Пункт чек-листа удалён"})
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
		Order:       int32(it.Order),
	}
}


func mapCommentToDTO(cm *domain.Comment) dto.CommentResponse {
	resp := dto.CommentResponse{
		ID:        uuid.MustParse(cm.ID),
		TaskID:    uuid.MustParse(cm.TaskID),
		AuthorID:  uuid.MustParse(cm.AuthorID),
		Content:   cm.Content,
		CreatedAt: cm.CreatedAt,
		UpdatedAt: cm.UpdatedAt,
	}
	if cm.ParentCommentID != nil {
		id := uuid.MustParse(*cm.ParentCommentID)
		resp.ParentCommentID = &id
	}
	return resp
}

func mapAttachmentToDTO(a *domain.Attachment) dto.AttachmentResponse {
	resp := dto.AttachmentResponse{
		ID:          uuid.MustParse(a.ID),
		FileName:    a.FileName,
		FilePath:    a.FilePath,
		FileSize:    a.FileSize,
		ContentType: a.ContentType,
		UploadedBy:  uuid.MustParse(a.UploadedBy),
		UploadedAt:  a.UploadedAt,
	}
	if a.TaskID != nil {
		id := uuid.MustParse(*a.TaskID)
		resp.TaskID = &id
	}
	if a.CommentID != nil {
		id := uuid.MustParse(*a.CommentID)
		resp.CommentID = &id
	}
	return resp
}

func mapTaskToDTO(t *domain.Task) dto.TaskResponse {
	projectID := uuid.MustParse(t.ProjectID)
	ownerMemberID := uuid.MustParse(t.OwnerID)
	var execMemberID *uuid.UUID
	if t.ExecutorID != nil {
		id := uuid.MustParse(*t.ExecutorID)
		execMemberID = &id
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

	tags := make([]dto.TagResponse, 0, len(t.Tags))
	for _, tag := range t.Tags {
		tags = append(tags, dto.TagResponse{
			ID:      tag.ID,
			BoardID: tag.BoardID,
			Name:    tag.Name,
		})
	}

	resp := dto.TaskResponse{
		ID:               uuid.MustParse(t.ID),
		Key:              t.Key,
		ProjectID:        projectID,
		BoardID:          uuid.MustParse(t.BoardID),
		OwnerMemberID:    ownerMemberID,
		ExecutorMemberID: execMemberID,
		Name:             t.Name,
		Description:      desc,
		Deadline:         deadline,
		ColumnID:         stringPtrToUUIDPtr(t.ColumnID),
		SwimlaneID:       swimlaneID,
		Priority:         t.Priority,
		Estimation:       t.Estimation,
		Progress:         progress,
		CreatedAt:        t.CreatedAt,
		ColumnName:       t.ColumnName,
		ColumnSystemType: t.ColumnSystemType,
		Tags:             tags,
	}

	resp.OwnerUserID = stringPtrToUUIDPtr(t.OwnerUserID)
	resp.ExecutorUserID = stringPtrToUUIDPtr(t.ExecutorUserID)

	return resp
}

func stringPtrToUUIDPtr(s *string) *uuid.UUID {
	if s == nil {
		return nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil
	}
	return &id
}

