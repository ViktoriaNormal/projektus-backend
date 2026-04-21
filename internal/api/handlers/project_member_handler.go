package handlers

import (
	"github.com/gin-gonic/gin"

	"projektus-backend/internal/api/dto"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/services"
)

type ProjectMemberHandler struct {
	service *services.ProjectMemberService
}

func NewProjectMemberHandler(service *services.ProjectMemberService) *ProjectMemberHandler {
	return &ProjectMemberHandler{service: service}
}

// toMemberRoleRefsDTO конвертирует domain-тип роли участника в DTO. Используется
// во всех местах ответа по участникам проекта — чтобы слои не ссылались друг
// на друга типами.
func toMemberRoleRefsDTO(rs []domain.ProjectMemberRoleRef) []dto.ProjectMemberRoleRef {
	if len(rs) == 0 {
		return nil
	}
	out := make([]dto.ProjectMemberRoleRef, len(rs))
	for i, r := range rs {
		out[i] = dto.ProjectMemberRoleRef{ID: r.ID, Name: r.Name}
	}
	return out
}

func (h *ProjectMemberHandler) ListMembers(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	members, err := h.service.ListMembers(c.Request.Context(), projectID)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось получить список участников")
		return
	}

	resp := make([]dto.ProjectMemberResponse, 0, len(members))
	for _, m := range members {
		resp = append(resp, dto.ProjectMemberResponse{
			ID:        m.ID,
			ProjectID: m.ProjectID,
			UserID:    m.UserID,
			Roles:     toMemberRoleRefsDTO(m.Roles),
		})
	}
	writeSuccess(c, resp)
}

func (h *ProjectMemberHandler) AddMember(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.AddMemberRequest](c)
	if !ok {
		return
	}

	member, err := h.service.AddMember(c.Request.Context(), projectID, req.UserID, req.Roles)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось добавить участника")
		return
	}

	writeSuccess(c, dto.ProjectMemberResponse{
		ID:        member.ID,
		ProjectID: member.ProjectID,
		UserID:    member.UserID,
		Roles:     toMemberRoleRefsDTO(member.Roles),
	})
}

func (h *ProjectMemberHandler) RemoveMember(c *gin.Context) {
	memberID, ok := paramUUID(c, "memberId")
	if !ok {
		return
	}

	if err := h.service.RemoveMember(c.Request.Context(), memberID); err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось удалить участника")
		return
	}

	writeSuccess(c, gin.H{"message": "Участник удален"})
}

func (h *ProjectMemberHandler) UpdateMemberRoles(c *gin.Context) {
	projectID, ok := paramUUID(c, "projectId")
	if !ok {
		return
	}
	memberID, ok := paramUUID(c, "memberId")
	if !ok {
		return
	}

	req, ok := bindJSON[dto.UpdateMemberRolesRequest](c)
	if !ok {
		return
	}

	member, err := h.service.UpdateMemberRoles(c.Request.Context(), memberID, projectID, req.Roles)
	if err != nil {
		if respondDomainErr(c, err) {
			return
		}
		respondInternal(c, err, "Не удалось обновить роли участника")
		return
	}

	writeSuccess(c, dto.ProjectMemberResponse{
		ID:        member.ID,
		ProjectID: member.ProjectID,
		UserID:    member.UserID,
		Roles:     toMemberRoleRefsDTO(member.Roles),
	})
}
