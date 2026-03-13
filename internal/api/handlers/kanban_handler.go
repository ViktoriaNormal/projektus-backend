package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type KanbanHandler struct {
	service *services.KanbanService
}

func NewKanbanHandler(service *services.KanbanService) *KanbanHandler {
	return &KanbanHandler{service: service}
}

// GetWipLimits возвращает WIP-лимиты проекта.
func (h *KanbanHandler) GetWipLimits(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	limits, err := h.service.GetWipLimits(c.Request.Context(), projectID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить WIP-лимиты")
		return
	}

	resp := make([]dto.WipLimitDTO, 0, len(limits))
	for _, l := range limits {
		resp = append(resp, mapWipLimitToDTO(&l))
	}
	writeSuccess(c, resp)
}

// UpdateWipLimits массово обновляет WIP-лимиты.
func (h *KanbanHandler) UpdateWipLimits(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор проекта")
		return
	}

	var req []dto.WipLimitDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return
	}

	limits := make([]domain.WipLimit, 0, len(req))
	for _, item := range req {
		l := domain.WipLimit{
			BoardID: item.BoardID,
			Limit:   item.Limit,
		}
		if item.ColumnID != nil {
			id := *item.ColumnID
			l.ColumnID = &id
		}
		if item.SwimlaneID != nil {
			id := *item.SwimlaneID
			l.SwimlaneID = &id
		}
		limits = append(limits, l)
	}

	if err := h.service.UpdateWipLimits(c.Request.Context(), projectID, limits); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось обновить WIP-лимиты")
		return
	}

	c.Status(http.StatusNoContent)
}

// GetCurrentWipCounts возвращает текущие WIP-счетчики по доске.
func (h *KanbanHandler) GetCurrentWipCounts(c *gin.Context) {
	boardIDStr := c.Param("boardId")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор доски")
		return
	}

	counts, err := h.service.GetCurrentWipCounts(c.Request.Context(), boardID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось получить WIP-счетчики")
		return
	}

	resp := make([]dto.WipCountDTO, 0, len(counts))
	for _, ctn := range counts {
		dtoItem := dto.WipCountDTO{
			Count: ctn.Count,
		}
		if ctn.ColumnID != nil {
			id := *ctn.ColumnID
			dtoItem.ColumnID = &id
		}
		if ctn.SwimlaneID != nil {
			id := *ctn.SwimlaneID
			dtoItem.SwimlaneID = &id
		}
		if ctn.Limit != nil {
			dtoItem.Limit = ctn.Limit
			if *ctn.Limit > 0 && ctn.Count > *ctn.Limit {
				dtoItem.Exceeded = true
			}
		}
		resp = append(resp, dtoItem)
	}

	writeSuccess(c, resp)
}

func mapWipLimitToDTO(l *domain.WipLimit) dto.WipLimitDTO {
	dtoItem := dto.WipLimitDTO{
		BoardID: l.BoardID,
	}
	if l.ColumnID != nil {
		id := *l.ColumnID
		dtoItem.ColumnID = &id
	}
	if l.SwimlaneID != nil {
		id := *l.SwimlaneID
		dtoItem.SwimlaneID = &id
	}
	if l.Limit != nil {
		dtoItem.Limit = l.Limit
	}
	return dtoItem
}

