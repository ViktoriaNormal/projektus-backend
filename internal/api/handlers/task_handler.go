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
	service       *services.TaskService
	boardSvc      *services.BoardService
	projectSvc    *services.ProjectService
	permissionSvc *services.PermissionService
}

func NewTaskHandler(service *services.TaskService, boardSvc *services.BoardService, projectSvc *services.ProjectService, permissionSvc *services.PermissionService) *TaskHandler {
	return &TaskHandler{service: service, boardSvc: boardSvc, projectSvc: projectSvc, permissionSvc: permissionSvc}
}

func (h *TaskHandler) SearchTasks(c *gin.Context) {
	req, ok := bindQuery[dto.SearchTasksRequest](c)
	if !ok {
		return
	}
	userID, ok := requireUserUUID(c)
	if !ok {
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

	// Семантика эндпоинта:
	//   * без project_id → персональная выборка «мои задачи»
	//     (автор / исполнитель / наблюдатель). Без исключений для админа —
	//     системные права не расширяют личный список.
	//   * с project_id → все задачи проекта при условии, что у пользователя
	//     есть доступ (участник проекта или system.projects.manage ≥ view).
	var tasks []domain.Task
	var err error
	if projectID == nil {
		tasks, err = h.service.SearchTasks(c.Request.Context(), userID, nil, columnID)
	} else {
		allowed, accessErr := h.permissionSvc.UserCanAccessProject(c.Request.Context(), userID, *projectID)
		if accessErr != nil {
			respondInternal(c, accessErr, "Не удалось проверить доступ к проекту")
			return
		}
		if !allowed {
			writeError(c, http.StatusForbidden, "ACCESS_DENIED", "Нет доступа к задачам проекта")
			return
		}
		tasks, err = h.service.SearchAllTasks(c.Request.Context(), projectID, columnID)
	}
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить список задач")
		return
	}
	resp := make([]dto.TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, mapTaskToDTO(&t))
	}
	writeSuccess(c, resp)
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	req, ok := bindJSON[dto.CreateTaskRequest](c)
	if !ok {
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
				columnID = col.ID
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
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать задачу: "+err.Error())
		return
	}
	// Re-fetch to get full data with user_id JOINs
	task, err := h.service.GetTask(c.Request.Context(), createdTask.ID)
	if err != nil {
		// Fallback: return created task without user_ids
		writeSuccess(c, mapTaskToDTO(createdTask))
		return
	}
	writeSuccess(c, mapTaskToDTO(task))
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	id, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	task, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить задачу")
		return
	}
	writeSuccess(c, mapTaskToDTO(task))
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
	id, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.UpdateTaskRequest](c)
	if !ok {
		return
	}
	task, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить задачу")
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
			v := req.ExecutorMemberID.Value
			task.ExecutorID = &v
		}
	}
	if req.ColumnID != nil {
		v := *req.ColumnID
		task.ColumnID = &v
	}
	if req.SwimlaneID.Set {
		if req.SwimlaneID.Null {
			task.SwimlaneID = nil
		} else {
			v := req.SwimlaneID.Value
			task.SwimlaneID = &v
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
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить задачу")
		return
	}
	writeSuccess(c, mapTaskToDTO(updated))
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
	id, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	if err := h.service.DeleteTask(c.Request.Context(), id); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить задачу")
		return
	}
	writeSuccess(c, gin.H{"message": "Задача удалена"})
}

func (h *TaskHandler) ListWatchers(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	watchers, err := h.service.ListWatchers(c.Request.Context(), taskID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить наблюдателей")
		return
	}
	resp := make([]dto.TaskWatcherResponse, 0, len(watchers))
	for _, w := range watchers {
		resp = append(resp, dto.TaskWatcherResponse{
			TaskID:   w.TaskID,
			MemberID: w.MemberID,
		})
	}
	writeSuccess(c, resp)
}

func (h *TaskHandler) AddWatcher(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.AddWatcherRequest](c)
	if !ok {
		return
	}
	if err := h.service.AddWatcher(c.Request.Context(), taskID, req.MemberID); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось добавить наблюдателя")
		return
	}
	writeSuccess(c, dto.TaskWatcherResponse{
		TaskID:   taskID,
		MemberID: req.MemberID,
	})
}

func (h *TaskHandler) RemoveWatcher(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	memberID, ok := paramUUID(c, "memberId")
	if !ok {
		return
	}
	if err := h.service.RemoveWatcher(c.Request.Context(), taskID, memberID); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить наблюдателя")
		return
	}
	writeSuccess(c, gin.H{"message": "Наблюдатель удалён"})
}

// Comments

func (h *TaskHandler) ListComments(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	comments, err := h.service.ListComments(c.Request.Context(), taskID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить комментарии")
		return
	}
	resp := make([]dto.CommentResponse, 0, len(comments))
	for _, cm := range comments {
		resp = append(resp, mapCommentToDTO(&cm))
	}
	writeSuccess(c, resp)
}

func (h *TaskHandler) CreateComment(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.CreateCommentRequest](c)
	if !ok {
		return
	}
	userID, ok := requireUserUUID(c)
	if !ok {
		return
	}
	comment, err := h.service.CreateComment(c.Request.Context(), taskID, userID, req.Content, req.ParentCommentID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать комментарий")
		return
	}
	writeSuccess(c, mapCommentToDTO(comment))
}

func (h *TaskHandler) DeleteComment(c *gin.Context) {
	commentID, ok := paramUUID(c, "commentId")
	if !ok {
		return
	}
	if err := h.service.DeleteComment(c.Request.Context(), commentID); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить комментарий")
		return
	}
	writeSuccess(c, gin.H{"message": "Комментарий удалён"})
}

// Attachments

func (h *TaskHandler) ListAttachments(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	attachments, err := h.service.ListAttachments(c.Request.Context(), taskID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить вложения")
		return
	}
	resp := make([]dto.AttachmentResponse, 0, len(attachments))
	for _, a := range attachments {
		resp = append(resp, mapAttachmentToDTO(&a))
	}
	writeSuccess(c, resp)
}

func (h *TaskHandler) UploadAttachment(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	userID, ok := requireUserUUID(c)
	if !ok {
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
		respondInternal(c, err, "Не удалось создать директорию для файлов")
		return
	}
	filePath := dir + "/" + uniqueName

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		respondInternal(c, err, "Не удалось сохранить файл")
		return
	}

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	attachment, err := h.service.CreateAttachment(c.Request.Context(), taskID, userID, file.Filename, filePath, contentType, file.Size)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось сохранить информацию о вложении")
		return
	}
	writeSuccess(c, mapAttachmentToDTO(attachment))
}

func (h *TaskHandler) DownloadAttachment(c *gin.Context) {
	attachmentID, ok := paramUUID(c, "attachmentId")
	if !ok {
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
	attachmentID, ok := paramUUID(c, "attachmentId")
	if !ok {
		return
	}
	if err := h.service.DeleteAttachment(c.Request.Context(), attachmentID); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить вложение")
		return
	}
	writeSuccess(c, gin.H{"message": "Вложение удалено"})
}

// Field values

func (h *TaskHandler) GetTaskFieldValues(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	fieldValues, err := h.service.GetTaskFieldValues(c.Request.Context(), taskID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить значения полей")
		return
	}
	resp := make([]dto.TaskFieldValueResponse, 0, len(fieldValues))
	for _, fv := range fieldValues {
		resp = append(resp, dto.TaskFieldValueResponse{
			FieldID:       fv.FieldID,
			ValueText:     fv.ValueText,
			ValueNumber:   fv.ValueNumber,
			ValueDatetime: fv.ValueDatetime,
		})
	}
	writeSuccess(c, resp)
}

func (h *TaskHandler) SetTaskFieldValue(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	fieldID, ok := paramUUID(c, "fieldId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.SetTaskFieldValueRequest](c)
	if !ok {
		return
	}
	if err := h.service.UpsertTaskFieldValue(c.Request.Context(), taskID, fieldID, req.ValueText, req.ValueNumber, req.ValueDatetime); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось сохранить значение поля")
		return
	}
	writeSuccess(c, gin.H{"message": "Значение поля сохранено"})
}

func (h *TaskHandler) ListDependencies(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	deps, err := h.service.ListDependencies(c.Request.Context(), taskID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить зависимости")
		return
	}
	resp := make([]dto.TaskDependencyResponse, 0, len(deps))
	for _, d := range deps {
		resp = append(resp, dto.TaskDependencyResponse{
			ID:              d.ID,
			TaskID:          d.TaskID,
			DependsOnTaskID: d.DependsOnTaskID,
			Type:            string(d.Type),
		})
	}
	writeSuccess(c, resp)
}

func (h *TaskHandler) AddDependency(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.AddDependencyRequest](c)
	if !ok {
		return
	}
	depType := domain.TaskDependencyType(req.Type)
	dep, err := h.service.AddDependency(c.Request.Context(), taskID, req.DependsOnTaskID, depType)
	if err != nil {
		// Сохраняем исторический маппинг: ErrConflict → VALIDATION_ERROR для зависимостей.
		if err == domain.ErrConflict {
			writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Связь между этими задачами уже существует")
			return
		}
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать зависимость")
		return
	}
	writeSuccess(c, dto.TaskDependencyResponse{
		ID:              dep.ID,
		TaskID:          dep.TaskID,
		DependsOnTaskID: dep.DependsOnTaskID,
		Type:            string(dep.Type),
	})
}

func (h *TaskHandler) RemoveDependency(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	dependencyID, ok := paramUUID(c, "dependencyId")
	if !ok {
		return
	}
	if err := h.service.RemoveDependency(c.Request.Context(), taskID, dependencyID); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить зависимость")
		return
	}
	writeSuccess(c, nil)
}

func (h *TaskHandler) ListChecklists(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	checklists, err := h.service.ListChecklistsWithItems(c.Request.Context(), taskID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить чек-листы")
		return
	}
	resp := make([]dto.ChecklistResponse, 0, len(checklists))
	for _, ch := range checklists {
		resp = append(resp, mapChecklistToDTO(&ch))
	}
	writeSuccess(c, resp)
}

func (h *TaskHandler) CreateChecklist(c *gin.Context) {
	taskID, ok := paramUUID(c, "taskId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.CreateChecklistRequest](c)
	if !ok {
		return
	}
	ch, err := h.service.CreateChecklist(c.Request.Context(), taskID, req.Name)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось создать чек-лист")
		return
	}
	writeSuccess(c, mapChecklistToDTO(ch))
}

func (h *TaskHandler) AddChecklistItem(c *gin.Context) {
	checklistID, ok := paramUUID(c, "checklistId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.CreateChecklistItemRequest](c)
	if !ok {
		return
	}
	item, err := h.service.AddChecklistItem(c.Request.Context(), checklistID, req.Content, int16(req.Order))
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось добавить пункт чек-листа")
		return
	}
	writeSuccess(c, mapChecklistItemToDTO(item))
}

func (h *TaskHandler) SetChecklistItemStatus(c *gin.Context) {
	itemID, ok := paramUUID(c, "itemId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.SetChecklistItemStatusRequest](c)
	if !ok {
		return
	}
	item, err := h.service.SetChecklistItemStatus(c.Request.Context(), itemID, *req.IsChecked)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить статус пункта чек-листа")
		return
	}
	writeSuccess(c, mapChecklistItemToDTO(item))
}

func (h *TaskHandler) UpdateChecklist(c *gin.Context) {
	checklistID, ok := paramUUID(c, "checklistId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.UpdateChecklistRequest](c)
	if !ok {
		return
	}
	ch, err := h.service.UpdateChecklistName(c.Request.Context(), checklistID, req.Name)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить чек-лист")
		return
	}
	writeSuccess(c, mapChecklistToDTO(ch))
}

func (h *TaskHandler) DeleteChecklist(c *gin.Context) {
	checklistID, ok := paramUUID(c, "checklistId")
	if !ok {
		return
	}
	if err := h.service.DeleteChecklist(c.Request.Context(), checklistID); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить чек-лист")
		return
	}
	writeSuccess(c, gin.H{"message": "Чек-лист удалён"})
}

func (h *TaskHandler) UpdateChecklistItem(c *gin.Context) {
	itemID, ok := paramUUID(c, "itemId")
	if !ok {
		return
	}
	req, ok := bindJSON[dto.UpdateChecklistItemRequest](c)
	if !ok {
		return
	}
	item, err := h.service.UpdateChecklistItemContent(c.Request.Context(), itemID, req.Content)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить пункт чек-листа")
		return
	}
	writeSuccess(c, mapChecklistItemToDTO(item))
}

func (h *TaskHandler) DeleteChecklistItem(c *gin.Context) {
	itemID, ok := paramUUID(c, "itemId")
	if !ok {
		return
	}
	if err := h.service.DeleteChecklistItem(c.Request.Context(), itemID); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить пункт чек-листа")
		return
	}
	writeSuccess(c, gin.H{"message": "Пункт чек-листа удалён"})
}

func mapChecklistToDTO(ch *domain.Checklist) dto.ChecklistResponse {
	taskID := ch.TaskID
	items := make([]dto.ChecklistItemResponse, 0, len(ch.Items))
	for _, it := range ch.Items {
		items = append(items, mapChecklistItemToDTO(&it))
	}
	return dto.ChecklistResponse{
		ID:     ch.ID,
		TaskID: taskID,
		Name:   ch.Name,
		Items:  items,
	}
}

func mapChecklistItemToDTO(it *domain.ChecklistItem) dto.ChecklistItemResponse {
	return dto.ChecklistItemResponse{
		ID:          it.ID,
		ChecklistID: it.ChecklistID,
		Content:     it.Content,
		IsChecked:   it.IsChecked,
		Order:       int32(it.Order),
	}
}

func mapCommentToDTO(cm *domain.Comment) dto.CommentResponse {
	resp := dto.CommentResponse{
		ID:        cm.ID,
		TaskID:    cm.TaskID,
		AuthorID:  cm.AuthorID,
		Content:   cm.Content,
		CreatedAt: cm.CreatedAt,
		UpdatedAt: cm.UpdatedAt,
	}
	if cm.ParentCommentID != nil {
		id := *cm.ParentCommentID
		resp.ParentCommentID = &id
	}
	return resp
}

func mapAttachmentToDTO(a *domain.Attachment) dto.AttachmentResponse {
	resp := dto.AttachmentResponse{
		ID:          a.ID,
		FileName:    a.FileName,
		FilePath:    a.FilePath,
		FileSize:    a.FileSize,
		ContentType: a.ContentType,
		UploadedBy:  a.UploadedBy,
		UploadedAt:  a.UploadedAt,
	}
	if a.TaskID != nil {
		id := *a.TaskID
		resp.TaskID = &id
	}
	if a.CommentID != nil {
		id := *a.CommentID
		resp.CommentID = &id
	}
	return resp
}

func mapTaskToDTO(t *domain.Task) dto.TaskResponse {
	projectID := t.ProjectID
	ownerMemberID := t.OwnerID
	var execMemberID *uuid.UUID
	if t.ExecutorID != nil {
		id := *t.ExecutorID
		execMemberID = &id
	}
	var desc *string
	if t.Description != nil {
		desc = t.Description
	}
	var swimlaneID *uuid.UUID
	if t.SwimlaneID != nil {
		id := *t.SwimlaneID
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
			ID:      tag.ID.String(),
			BoardID: tag.BoardID.String(),
			Name:    tag.Name,
		})
	}

	var columnID *uuid.UUID
	if t.ColumnID != nil {
		v := *t.ColumnID
		columnID = &v
	}

	resp := dto.TaskResponse{
		ID:               t.ID,
		Key:              t.Key,
		ProjectID:        projectID,
		BoardID:          t.BoardID,
		OwnerMemberID:    ownerMemberID,
		ExecutorMemberID: execMemberID,
		Name:             t.Name,
		Description:      desc,
		Deadline:         deadline,
		ColumnID:         columnID,
		SwimlaneID:       swimlaneID,
		Priority:         t.Priority,
		Estimation:       t.Estimation,
		Progress:         progress,
		CreatedAt:        t.CreatedAt,
		ColumnName:       t.ColumnName,
		ColumnSystemType: t.ColumnSystemType,
		Tags:             tags,
	}

	if t.OwnerUserID != nil {
		v := *t.OwnerUserID
		resp.OwnerUserID = &v
	}
	if t.ExecutorUserID != nil {
		v := *t.ExecutorUserID
		resp.ExecutorUserID = &v
	}

	return resp
}
