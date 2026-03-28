package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"projektus-backend/internal/db"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
)

const ProjectAdminRoleName = "Администратор проекта"

type ProjectService struct {
	repo            repositories.ProjectRepository
	templateSvc     *TemplateService
	boardRepo       repositories.BoardRepository
	projectRoleRepo repositories.ProjectRoleRepository
	projectParamRepo repositories.ProjectParamRepository
	memberRepo      repositories.ProjectMemberRepository
}

func NewProjectService(
	repo repositories.ProjectRepository,
	templateSvc *TemplateService,
	boardRepo repositories.BoardRepository,
	projectRoleRepo repositories.ProjectRoleRepository,
	projectParamRepo repositories.ProjectParamRepository,
	memberRepo repositories.ProjectMemberRepository,
) *ProjectService {
	return &ProjectService{
		repo:             repo,
		templateSvc:      templateSvc,
		boardRepo:        boardRepo,
		projectRoleRepo:  projectRoleRepo,
		projectParamRepo: projectParamRepo,
		memberRepo:       memberRepo,
	}
}

func (s *ProjectService) CreateProject(ctx context.Context, ownerID uuid.UUID, name, description, projectType string) (*domain.Project, error) {
	if name == "" {
		return nil, domain.ErrInvalidInput
	}

	pt := domain.ProjectType(strings.ToLower(projectType))
	if pt != domain.ProjectTypeScrum && pt != domain.ProjectTypeKanban {
		return nil, domain.ErrInvalidInput
	}

	keyPrefix := generateProjectKeyPrefix(name)
	key, err := s.generateUniqueProjectKey(ctx, keyPrefix)
	if err != nil {
		return nil, err
	}

	var descPtr *string
	if description != "" {
		descPtr = &description
	}

	p := &domain.Project{
		Key:         key,
		Name:        name,
		Description: descPtr,
		Type:        pt,
		OwnerID:     ownerID,
		Status:      domain.ProjectStatusActive,
	}

	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return nil, err
	}

	// Copy template structure
	if s.templateSvc != nil {
		if err := s.copyTemplateToProject(ctx, created.ID, string(pt)); err != nil {
			// Best effort — don't fail project creation if template not found
			_ = err
		}
	}

	// Create mandatory "Администратор проекта" role
	adminRole, err := s.createProjectAdminRole(ctx, created.ID, string(pt))
	if err != nil {
		return nil, err
	}

	// Add creator as first member with admin role
	if s.memberRepo != nil {
		member, err := s.memberRepo.AddMember(ctx, created.ID, ownerID)
		if err != nil {
			return nil, err
		}
		adminRoleID := uuid.MustParse(adminRole.ID)
		if err := s.memberRepo.ReplaceMemberRoles(ctx, member.ID, []uuid.UUID{adminRoleID}); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *ProjectService) copyTemplateToProject(ctx context.Context, projectID uuid.UUID, projectType string) error {
	tmpl, data, err := s.templateSvc.GetByType(ctx, projectType)
	if err != nil {
		return err
	}
	_ = tmpl

	pid := projectID.String()

	// Copy boards with all nested entities
	for _, tb := range data.Boards {
		board, err := s.boardRepo.CreateBoard(ctx, &domain.Board{
			ProjectID:       &pid,
			Name:            tb.Name,
			Description:     &tb.Description,
			Order:           int16(tb.Order),
			PriorityType:    tb.PriorityType,
			EstimationUnit:  tb.EstimationUnit,
			SwimlaneGroupBy: tb.SwimlaneGroupBy,
		})
		if err != nil {
			return err
		}

		// Copy columns
		for _, tc := range tb.Columns {
			var wl *int16
			if tc.WipLimit != nil {
				v := int16(*tc.WipLimit)
				wl = &v
			}
			st := domain.SystemStatusType(tc.SystemType)
			col, err := s.boardRepo.CreateColumn(ctx, &domain.Column{
				BoardID:    board.ID,
				Name:       tc.Name,
				SystemType: &st,
				WipLimit:   wl,
				Order:      int16(tc.Order),
				IsLocked:   tc.IsLocked,
			})
			if err != nil {
				return err
			}
			// Copy column note
			if tc.Note != nil && *tc.Note != "" {
				cid := col.ID
				_, _ = s.boardRepo.CreateNoteForColumn(ctx, &domain.Note{
					ColumnID: &cid,
					Content:  *tc.Note,
				})
			}
		}

		// Copy swimlanes
		for _, ts := range tb.Swimlanes {
			var wl *int16
			if ts.WipLimit != nil {
				v := int16(*ts.WipLimit)
				wl = &v
			}
			sw, err := s.boardRepo.CreateSwimlane(ctx, &domain.Swimlane{
				BoardID:  board.ID,
				Name:     ts.Name,
				WipLimit: wl,
				Order:    int16(ts.Order),
			})
			if err != nil {
				return err
			}
			// Copy swimlane note
			if ts.Note != nil && *ts.Note != "" {
				sid := sw.ID
				_, _ = s.boardRepo.CreateNoteForSwimlane(ctx, &domain.Note{
					SwimlaneID: &sid,
					Content:    *ts.Note,
				})
			}
		}

		// Copy custom fields
		for _, cf := range tb.CustomFields {
			_, _ = s.boardRepo.CreateCustomField(ctx, &domain.BoardCustomField{
				BoardID:    board.ID,
				Name:       cf.Name,
				FieldType:  cf.FieldType,
				IsSystem:   cf.IsSystem,
				IsRequired: cf.IsRequired,
				Order:      cf.Order,
				Options:    cf.Options,
			})
		}
	}

	// Copy project params
	nullPID := uuid.NullUUID{UUID: projectID, Valid: true}
	for _, tp := range data.Params {
		_, _ = s.projectParamRepo.Create(ctx, db.CreateProjectParamParams{
			ProjectID:  nullPID,
			Name:       tp.Name,
			FieldType:  tp.FieldType,
			IsSystem:   false,
			IsRequired: tp.IsRequired,
			SortOrder:  tp.Order,
			Options:    repositories.OptionsToJSON(tp.Options),
			Value:      sql.NullString{},
		})
	}

	// Copy roles (except admin — that's created separately)
	for _, tr := range data.Roles {
		role, err := s.projectRoleRepo.Create(ctx, db.CreateProjRoleDefinitionParams{
			ProjectID:   nullPID,
			Name:        tr.Name,
			Description: tr.Description,
			IsAdmin:     false,
		})
		if err != nil {
			continue
		}
		roleID := uuid.MustParse(role.ID)
		for _, p := range tr.Permissions {
			_ = s.projectRoleRepo.UpsertPermission(ctx, roleID, p.Area, p.Access)
		}
	}

	return nil
}

func (s *ProjectService) createProjectAdminRole(ctx context.Context, projectID uuid.UUID, projectType string) (*domain.ProjectRole, error) {
	// Get all permission areas for this project type
	allAreas := getPermissionAreasForType(projectType)

	role, err := s.projectRoleRepo.Create(ctx, db.CreateProjRoleDefinitionParams{
		ProjectID:   uuid.NullUUID{UUID: projectID, Valid: true},
		Name:        ProjectAdminRoleName,
		Description: "Полный доступ ко всем сущностям проекта",
		IsAdmin:     true,
	})
	if err != nil {
		return nil, err
	}

	roleID := uuid.MustParse(role.ID)
	perms := make([]domain.ProjectRolePermission, 0, len(allAreas))
	for _, area := range allAreas {
		_ = s.projectRoleRepo.UpsertPermission(ctx, roleID, area, "full")
		perms = append(perms, domain.ProjectRolePermission{Area: area, Access: "full"})
	}
	role.Permissions = perms
	return role, nil
}

func getPermissionAreasForType(projectType string) []string {
	if projectType == "scrum" {
		return []string{"sprints", "boards", "analytics", "backlog", "tasks", "project_settings"}
	}
	// kanban
	return []string{"boards", "wip_limits", "analytics", "tasks", "project_settings"}
}

func (s *ProjectService) GetProject(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProjectService) ListProjects(ctx context.Context, userID uuid.UUID, query *string, status, projectType *string) ([]domain.Project, error) {
	return s.repo.ListUserProjects(ctx, userID, query, status, projectType)
}

func (s *ProjectService) UpdateProject(ctx context.Context, p *domain.Project) (*domain.Project, error) {
	return s.repo.Update(ctx, p)
}

func (s *ProjectService) DeleteProject(ctx context.Context, id uuid.UUID, confirm bool) error {
	if !confirm {
		return domain.ErrInvalidInput
	}
	return s.repo.Delete(ctx, id)
}

func (s *ProjectService) generateUniqueProjectKey(ctx context.Context, prefix string) (string, error) {
	for i := 1; i < 1000000; i++ {
		key := fmt.Sprintf("%s-%d", prefix, i)
		_, err := s.repo.GetByKey(ctx, key)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return key, nil
			}
			return "", err
		}
	}
	return "", domain.ErrConflict
}

// cyrToLat — таблица транслитерации кириллицы в латиницу.
var cyrToLat = map[rune]string{
	'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ё': "E",
	'Ж': "ZH", 'З': "Z", 'И': "I", 'Й': "Y", 'К': "K", 'Л': "L", 'М': "M",
	'Н': "N", 'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'У': "U",
	'Ф': "F", 'Х': "KH", 'Ц': "TS", 'Ч': "CH", 'Ш': "SH", 'Щ': "SHCH",
	'Ъ': "", 'Ы': "Y", 'Ь': "", 'Э': "E", 'Ю': "YU", 'Я': "YA",
}

// transliterate переводит строку из кириллицы в латиницу. Латинские символы остаются как есть.
func transliterate(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if lat, ok := cyrToLat[r]; ok {
			b.WriteString(lat)
		} else if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func generateProjectKeyPrefix(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "PRJ"
	}

	transliterated := transliterate(trimmed)
	if transliterated == "" {
		return "PRJ"
	}

	// Берём первые 3-5 латинских символов (только буквы)
	var letters []rune
	for _, r := range transliterated {
		if r >= 'A' && r <= 'Z' {
			letters = append(letters, r)
		}
		if len(letters) >= 4 {
			break
		}
	}
	if len(letters) == 0 {
		return "PRJ"
	}
	return string(letters)
}
