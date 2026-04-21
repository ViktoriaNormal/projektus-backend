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


type ProjectService struct {
	repo                 repositories.ProjectRepository
	templateSvc          *TemplateService
	boardRepo            repositories.BoardRepository
	columnRepo           repositories.ColumnRepository
	swimlaneRepo         repositories.SwimlaneRepository
	noteRepo             repositories.NoteRepository
	boardCustomFieldRepo repositories.BoardCustomFieldRepository
	projectRoleRepo      repositories.ProjectRoleRepository
	projectParamRepo     repositories.ProjectParamRepository
	memberRepo           repositories.ProjectMemberRepository
	conn                 *sql.DB
}

func NewProjectService(
	repo repositories.ProjectRepository,
	templateSvc *TemplateService,
	boardRepo repositories.BoardRepository,
	columnRepo repositories.ColumnRepository,
	swimlaneRepo repositories.SwimlaneRepository,
	noteRepo repositories.NoteRepository,
	boardCustomFieldRepo repositories.BoardCustomFieldRepository,
	projectRoleRepo repositories.ProjectRoleRepository,
	projectParamRepo repositories.ProjectParamRepository,
	memberRepo repositories.ProjectMemberRepository,
	conn *sql.DB,
) *ProjectService {
	return &ProjectService{
		repo:                 repo,
		templateSvc:          templateSvc,
		boardRepo:            boardRepo,
		columnRepo:           columnRepo,
		swimlaneRepo:         swimlaneRepo,
		noteRepo:             noteRepo,
		boardCustomFieldRepo: boardCustomFieldRepo,
		projectRoleRepo:      projectRoleRepo,
		projectParamRepo:     projectParamRepo,
		memberRepo:           memberRepo,
		conn:                 conn,
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

	var sprintDuration *int
	if pt == domain.ProjectTypeScrum {
		d := 2
		sprintDuration = &d
	}

	p := &domain.Project{
		Key:                 key,
		Name:                name,
		Description:         descPtr,
		Type:                pt,
		OwnerID:             ownerID,
		Status:              domain.ProjectStatusActive,
		SprintDurationWeeks: sprintDuration,
	}

	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return nil, err
	}

	// Copy template structure (boards, params, roles)
	var adminRoleID uuid.UUID
	if s.templateSvc != nil {
		rid, err := s.copyTemplateToProject(ctx, created.ID, string(pt))
		if err != nil {
			_ = err
		} else {
			adminRoleID = rid
		}
	}

	// Add creator as first member with admin role
	if s.memberRepo != nil && adminRoleID != uuid.Nil {
		member, err := s.memberRepo.AddMember(ctx, created.ID, ownerID)
		if err != nil {
			return nil, err
		}
		if err := s.memberRepo.ReplaceMemberRoles(ctx, member.ID, []uuid.UUID{adminRoleID}); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *ProjectService) copyTemplateToProject(ctx context.Context, projectID uuid.UUID, projectType string) (uuid.UUID, error) {
	tmpl, data, err := s.templateSvc.GetByType(ctx, projectType)
	if err != nil {
		return uuid.Nil, err
	}
	_ = tmpl

	pid := projectID

	// Copy boards with all nested entities
	for _, tb := range data.Boards {
		board, err := s.boardRepo.CreateBoard(ctx, &domain.Board{
			ProjectID:       &pid,
			Name:            tb.Name,
			Description:     &tb.Description,
			IsDefault:       tb.IsDefault,
			Order:           int16(tb.Order),
			PriorityType:    tb.PriorityType,
			EstimationUnit:  tb.EstimationUnit,
			SwimlaneGroupBy: tb.SwimlaneGroupBy,
			PriorityOptions: tb.PriorityOptions,
		})
		if err != nil {
			return uuid.Nil, err
		}

		// Copy columns
		for _, tc := range tb.Columns {
			var wl *int16
			if tc.WipLimit != nil {
				v := int16(*tc.WipLimit)
				wl = &v
			}
			st := domain.SystemStatusType(tc.SystemType)
			col, err := s.columnRepo.Create(ctx, &domain.Column{
				BoardID:    board.ID,
				Name:       tc.Name,
				SystemType: &st,
				WipLimit:   wl,
				Order:      int16(tc.Order),
				IsLocked:   tc.IsLocked,
			})
			if err != nil {
				return uuid.Nil, err
			}
			// Copy column note
			if tc.Note != nil && *tc.Note != "" {
				cid := col.ID
				_, _ = s.noteRepo.CreateForColumn(ctx, &domain.Note{
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
			sw, err := s.swimlaneRepo.Create(ctx, &domain.Swimlane{
				BoardID:  board.ID,
				Name:     ts.Name,
				WipLimit: wl,
				Order:    int16(ts.Order),
			})
			if err != nil {
				return uuid.Nil, err
			}
			// Copy swimlane note
			if ts.Note != nil && *ts.Note != "" {
				sid := sw.ID
				_, _ = s.noteRepo.CreateForSwimlane(ctx, &domain.Note{
					SwimlaneID: &sid,
					Content:    *ts.Note,
				})
			}
		}

		// Copy custom fields and build ID mapping for swimlane_group_by remapping
		fieldIDMap := make(map[string]string) // template field ID → project field ID
		for _, cf := range tb.CustomFields {
			newField, err := s.boardCustomFieldRepo.Create(ctx, &domain.BoardCustomField{
				BoardID:    board.ID,
				Name:       cf.Name,
				FieldType:  cf.FieldType,
				IsSystem:   cf.IsSystem,
				IsRequired: cf.IsRequired,
				Options:    cf.Options,
			})
			if err == nil && newField != nil {
				fieldIDMap[cf.ID.String()] = newField.ID.String()
			}
		}

		// Remap swimlane_group_by from template field ID to project field ID
		if tb.SwimlaneGroupBy != "" {
			if newFieldID, ok := fieldIDMap[tb.SwimlaneGroupBy]; ok {
				board.SwimlaneGroupBy = newFieldID
				_, _ = s.boardRepo.UpdateBoard(ctx, board)
			}
		}
	}

	// Copy project params
	nullPID := uuid.NullUUID{UUID: projectID, Valid: true}
	for _, tp := range data.Params {
		_, _ = s.projectParamRepo.Create(ctx, db.CreateProjectParamParams{
			ProjectID:  nullPID,
			Name:       tp.Name,
			FieldType:  tp.FieldType,
			IsRequired: tp.IsRequired,
			Options:    repositories.OptionsToJSON(tp.Options),
			Value:      sql.NullString{},
		})
	}

	// Copy all roles from template (including admin)
	var adminRoleID uuid.UUID
	for _, tr := range data.Roles {
		role, err := s.projectRoleRepo.Create(ctx, db.CreateProjRoleDefinitionParams{
			ProjectID:   nullPID,
			Name:        tr.Name,
			Description: tr.Description,
			IsAdmin:     tr.IsAdmin,
		})
		if err != nil {
			continue
		}
		roleID := role.ID
		for _, p := range tr.Permissions {
			_ = s.projectRoleRepo.UpsertPermission(ctx, roleID, p.Area, p.Access)
		}
		if tr.IsAdmin {
			adminRoleID = roleID
		}
	}

	return adminRoleID, nil
}


func (s *ProjectService) GetProject(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProjectService) ListProjects(ctx context.Context, userID uuid.UUID, query *string, status, projectType *string) ([]domain.Project, error) {
	return s.repo.ListUserProjects(ctx, userID, query, status, projectType)
}

func (s *ProjectService) ListAllProjects(ctx context.Context, query *string, status, projectType *string) ([]domain.Project, error) {
	return s.repo.ListAllProjects(ctx, query, status, projectType)
}

func (s *ProjectService) UpdateProject(ctx context.Context, p *domain.Project, newOwnerID *uuid.UUID) (*domain.Project, error) {
	// Каскад owner → project_members завязан на ту же транзакцию, что и апдейт
	// самого проекта: если добавить участника/роль не удалось, изменение
	// owner_id в projects тоже откатывается — иначе на проекте остался бы
	// ответственный, которого нет среди участников (именно так и выглядел
	// воспроизведённый баг).
	return repositories.InTxT(ctx, s.conn, func(qtx *db.Queries) (*domain.Project, error) {
		txProjectRepo := repositories.NewProjectRepository(qtx)

		current, err := txProjectRepo.GetByID(ctx, p.ID)
		if err != nil {
			return nil, err
		}

		updated, err := txProjectRepo.Update(ctx, p)
		if err != nil {
			return nil, err
		}

		// Каскад срабатывает только если owner действительно меняется.
		// Если клиент прислал owner_id, равный текущему — это no-op, трогать
		// project_members не нужно (и не хотим случайно перетасовать роли).
		if newOwnerID != nil && *newOwnerID != current.OwnerID {
			if err := s.ensureOwnerIsMemberTx(ctx, qtx, p.ID, *newOwnerID); err != nil {
				return nil, err
			}
		}

		return updated, nil
	})
}

// ensureOwnerIsMemberTx — добавляет пользователя в project_members и
// гарантирует, что у него есть admin-роль проекта. Существующие роли участника
// сохраняются (AddRoleToMember идемпотентен через ON CONFLICT DO NOTHING).
// Если в проекте нет admin-роли — возвращает ErrProjectAdminRoleMissing,
// транзакция откатывается.
func (s *ProjectService) ensureOwnerIsMemberTx(ctx context.Context, qtx *db.Queries, projectID, ownerID uuid.UUID) error {
	memberRepo := repositories.NewProjectMemberRepository(qtx)
	roleRepo := repositories.NewProjectRoleRepository(qtx)

	member, err := memberRepo.GetByProjectAndUser(ctx, projectID, ownerID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	if member == nil {
		member, err = memberRepo.AddMember(ctx, projectID, ownerID)
		if err != nil {
			return err
		}
	}

	adminRoleID, err := roleRepo.GetProjectAdminRoleID(ctx, projectID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrProjectAdminRoleMissing
		}
		return err
	}

	// Добавляем admin-роль, сохраняя уже назначенные роли участника
	// (Разработчик/Менеджер и т.п.). Политика «старый владелец сохраняет
	// admin-роль» сознательная: смена ответственного не должна молча лишать
	// прежнего прав.
	return memberRepo.AddRoleToMember(ctx, member.ID, adminRoleID)
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
