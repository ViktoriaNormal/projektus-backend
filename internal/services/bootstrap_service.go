package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/google/uuid"

	"projektus-backend/config"
	"projektus-backend/internal/domain"
	"projektus-backend/internal/repositories"
	"projektus-backend/pkg/errctx"
)

// BootstrapService отвечает за одноразовую инициализацию системы
// при первом развёртывании: системная роль «Администратор системы»,
// первый администратор, начальная парольная политика, дефолтные шаблоны
// проектов Scrum и Kanban.
type BootstrapService struct {
	cfg              *config.Config
	userRepo         repositories.UserRepository
	roleRepo         repositories.RoleRepository
	passwordPolicyRp repositories.PasswordPolicyRepository
	templateSvc      *TemplateService
	passwordSvc      PasswordService
	notificationSvc  NotificationService
}

func NewBootstrapService(
	cfg *config.Config,
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
	passwordPolicyRp repositories.PasswordPolicyRepository,
	templateSvc *TemplateService,
	passwordSvc PasswordService,
	notificationSvc NotificationService,
) *BootstrapService {
	return &BootstrapService{
		cfg:              cfg,
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		passwordPolicyRp: passwordPolicyRp,
		templateSvc:      templateSvc,
		passwordSvc:      passwordSvc,
		notificationSvc:  notificationSvc,
	}
}

// Имена стандартных системных ролей, гарантированно существующих в системе.
// Используются для идемпотентного bootstrap и для назначения дефолтной роли
// пользователям, создаваемым через публичную регистрацию.
const (
	SystemRoleNameAdmin           = "Администратор системы"
	SystemRoleNameUserManager     = "Менеджер пользователей"
	SystemRoleNameTemplateManager = "Менеджер шаблонов проектов"
	SystemRoleNameProjectManager  = "Менеджер проектов"
	SystemRoleNameRegularUser     = "Обычный пользователь"
)

// EnsureInitialState — точка входа первичной инициализации системы.
// Идемпотентна: каждый шаг проверяет, существует ли нужная запись,
// и создаёт её только при отсутствии.
//
// Выполняемые шаги:
//  1. ensureSystemRoles — создать системную роль «Администратор системы»,
//     а также набор стандартных системных ролей (менеджеры и «Обычный
//     пользователь») для идемпотентности bootstrap на чистой БД.
//  2. ensurePasswordPolicy — создать начальную парольную политику,
//     если в таблице нет ни одной записи.
//  3. ensureDefaultTemplates — создать шаблоны Scrum и Kanban по умолчанию,
//     если их нет в БД.
//  4. ensureInitialAdminUser — создать первого администратора и привязать
//     к нему роль, если нет ни одного активного пользователя с правом
//     system.users.manage=full.
func (s *BootstrapService) EnsureInitialState(ctx context.Context) error {
	role, err := s.ensureSystemRoles(ctx)
	if err != nil {
		return err
	}

	if err := s.ensurePasswordPolicy(ctx); err != nil {
		return err
	}

	if err := s.ensureDefaultTemplates(ctx); err != nil {
		return err
	}

	return s.ensureInitialAdminUser(ctx, role)
}

// ensureSystemRoles гарантирует, что в системе есть стандартный набор
// системных ролей. Возвращает роль «Администратор системы» для последующей
// привязки к первому администратору.
//
// Состав ролей:
//   - «Администратор системы» — все системные права `full`, is_admin=true.
//   - «Менеджер пользователей» — только `system.users.manage=full`.
//   - «Менеджер шаблонов проектов» — только `system.project_templates.manage=full`.
//   - «Менеджер проектов» — только `system.projects.manage=full`.
//   - «Обычный пользователь» — без системных прав (роль-заглушка,
//     чтобы ни один пользователь не оставался без роли).
//
// Идемпотентна: если роль с нужным именем уже есть, bootstrap её использует.
func (s *BootstrapService) ensureSystemRoles(ctx context.Context) (*domain.Role, error) {
	adminRole, err := s.ensureAdminRole(ctx)
	if err != nil {
		return nil, err
	}

	nonAdminSpecs := []struct {
		name        string
		description string
		permission  string
	}{
		{SystemRoleNameUserManager, "Управление учётными записями пользователей системы", "system.users.manage"},
		{SystemRoleNameTemplateManager, "Создание и настройка шаблонов проектов Scrum/Kanban", "system.project_templates.manage"},
		{SystemRoleNameProjectManager, "Просмотр и управление всеми проектами в системе", "system.projects.manage"},
		{SystemRoleNameRegularUser, "Базовый уровень доступа без системных привилегий", ""},
	}
	for _, spec := range nonAdminSpecs {
		if _, err := s.ensureNonAdminSystemRole(ctx, spec.name, spec.description, spec.permission); err != nil {
			return nil, errctx.Wrap(err, "ensureSystemRoles", "role", spec.name)
		}
	}

	return adminRole, nil
}

// ensureAdminRole возвращает существующую системную роль администратора
// или создаёт новую «Администратор системы» со всеми системными правами `full`.
func (s *BootstrapService) ensureAdminRole(ctx context.Context) (*domain.Role, error) {
	role, err := s.roleRepo.GetSystemAdminRole(ctx)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, errctx.Wrap(err, "ensureAdminRole")
	}
	if role == nil {
		role, err = s.roleRepo.CreateAdminSystemRole(ctx, SystemRoleNameAdmin,
			"Полный доступ ко всем функциям системы")
		if err != nil {
			return nil, errctx.Wrap(err, "ensureAdminRole")
		}
		log.Printf("[Bootstrap] system role created: %s", role.Name)
	}

	for _, p := range repositories.AllPermissions {
		if p.Scope != "system" {
			continue
		}
		if err := s.roleRepo.AddPermissionToRole(ctx, role.ID, p.Code, "full"); err != nil {
			return nil, errctx.Wrap(err, "ensureAdminRole", "permission", p.Code)
		}
	}
	return role, nil
}

// ensureNonAdminSystemRole создаёт (или возвращает существующую) системную роль
// с указанным именем и единственным правом access=full (если permissionCode не
// пуст). Ищет по имени — сначала в ListSystemRoles, чтобы не плодить дубли.
func (s *BootstrapService) ensureNonAdminSystemRole(ctx context.Context, name, description, permissionCode string) (*domain.Role, error) {
	roles, err := s.roleRepo.ListSystemRoles(ctx)
	if err != nil {
		return nil, err
	}
	var role *domain.Role
	for i := range roles {
		if roles[i].Name == name {
			role = &roles[i]
			break
		}
	}
	if role == nil {
		created, err := s.roleRepo.CreateSystemRole(ctx, name, description)
		if err != nil {
			return nil, err
		}
		role = created
		log.Printf("[Bootstrap] system role created: %s", role.Name)
	}
	if permissionCode != "" {
		if err := s.roleRepo.AddPermissionToRole(ctx, role.ID, permissionCode, "full"); err != nil {
			return nil, err
		}
	}
	return role, nil
}

// ensurePasswordPolicy создаёт начальную парольную политику, если её нет:
// MinLength=8, все require_* = true, notes = NULL.
func (s *BootstrapService) ensurePasswordPolicy(ctx context.Context) error {
	_, err := s.passwordPolicyRp.GetCurrent(ctx)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return errctx.Wrap(err, "ensurePasswordPolicy")
	}
	if _, err := s.passwordPolicyRp.Insert(ctx, 8, true, true, true, true, nil, nil); err != nil {
		return errctx.Wrap(err, "ensurePasswordPolicy")
	}
	log.Printf("[Bootstrap] default password policy created (min_length=8, all requirements enabled)")
	return nil
}

// ensureDefaultTemplates создаёт дефолтные шаблоны Scrum и Kanban,
// если их ещё нет в БД. Создаёт:
//   - шаблон (templates) + основную доску с системными колонками
//     (и дорожками-классами обслуживания для Kanban);
//   - роль «Администратор проекта» со всеми проектными правами `full` —
//     без неё при создании проекта некому было бы назначить владельца
//     (см. ProjectService.copyTemplateToProject).
//
// Системные параметры проекта (Название, Описание, Статус, Ответственный,
// Дата создания) и системные поля задач в БД не вставляются: они
// определены как Go-константы в internal/domain/system_fields.go и
// добавляются к ответу API в runtime через GenerateSystemProjectParams /
// GenerateSystemBoardFields (см. handlers/template_handler.go
// injectSystemFields).
func (s *BootstrapService) ensureDefaultTemplates(ctx context.Context) error {
	specs := []struct {
		name        string
		description string
		projectType string
	}{
		{"Scrum стандартный", "Стандартный шаблон для Scrum-проектов", "scrum"},
		{"Kanban стандартный", "Стандартный шаблон для Kanban-проектов", "kanban"},
	}
	for _, sp := range specs {
		if _, _, err := s.templateSvc.GetByType(ctx, sp.projectType); err == nil {
			continue
		} else if !errors.Is(err, domain.ErrNotFound) {
			return errctx.Wrap(err, "ensureDefaultTemplates", "projectType", sp.projectType)
		}
		desc := sp.description
		tmpl, _, err := s.templateSvc.Create(ctx, sp.name, &desc, sp.projectType)
		if err != nil {
			return errctx.Wrap(err, "ensureDefaultTemplates", "projectType", sp.projectType)
		}

		if err := s.seedTemplateStandardRoles(ctx, tmpl.ID, sp.projectType); err != nil {
			return errctx.Wrap(err, "ensureDefaultTemplates", "projectType", sp.projectType)
		}
		log.Printf("[Bootstrap] default %s template created: %s", sp.projectType, sp.name)
	}
	return nil
}

// seedTemplateStandardRoles создаёт в шаблоне стандартный набор проектных ролей:
// «Администратор проекта» (все применимые права `full`, is_admin=true) плюс
// пять функциональных ролей (Менеджер проекта, Разработчик, Тестировщик,
// Аналитик, Наблюдатель). is_admin не выставляется для функциональных ролей —
// только CreateRole умеет создавать admin-роль, поэтому первым создаём её
// напрямую через низкоуровневый метод шаблонного сервиса.
func (s *BootstrapService) seedTemplateStandardRoles(ctx context.Context, templateID uuid.UUID, projectType string) error {
	adminPerms := make([]domain.TemplateRolePermission, 0, len(repositories.ProjectPermissionAreas))
	for _, area := range repositories.ProjectPermissionAreas {
		if !isAreaAvailable(area.AvailableFor, projectType) {
			continue
		}
		adminPerms = append(adminPerms, domain.TemplateRolePermission{Area: area.Area, Access: "full"})
	}
	if _, err := s.templateSvc.CreateRole(ctx, templateID,
		"Администратор проекта", "Полный доступ ко всем сущностям проекта", adminPerms); err != nil {
		return err
	}

	type roleSpec struct {
		name, description string
		access            map[string]string // area → access level
	}

	// Базовый набор access по каждой роли. Права project.sprints
	// применяются только для Scrum — на Kanban они фильтруются ниже.
	specs := []roleSpec{
		{"Менеджер проекта", "Управление задачами, спринтами и настройками проекта", map[string]string{
			"project.boards": "full", "project.tasks": "full", "project.sprints": "full",
			"project.settings": "full", "project.members": "full", "project.roles": "view",
			"project.analytics": "full",
		}},
		{"Разработчик", "Работа с задачами и спринтами", map[string]string{
			"project.boards": "view", "project.tasks": "full", "project.sprints": "full",
			"project.settings": "view", "project.members": "view", "project.analytics": "view",
		}},
		{"Тестировщик", "Проверка задач и работа с багами", map[string]string{
			"project.boards": "view", "project.tasks": "full", "project.sprints": "view",
			"project.settings": "view", "project.members": "view", "project.analytics": "view",
		}},
		{"Аналитик", "Анализ метрик и описание задач", map[string]string{
			"project.boards": "view", "project.tasks": "view", "project.sprints": "view",
			"project.settings": "view", "project.members": "view", "project.analytics": "full",
		}},
		{"Наблюдатель", "Только просмотр данных проекта", map[string]string{
			"project.boards": "view", "project.tasks": "view", "project.sprints": "view",
			"project.settings": "view", "project.members": "view", "project.analytics": "view",
		}},
	}

	for _, spec := range specs {
		perms := make([]domain.TemplateRolePermission, 0, len(spec.access))
		for _, area := range repositories.ProjectPermissionAreas {
			if !isAreaAvailable(area.AvailableFor, projectType) {
				continue
			}
			if acc, ok := spec.access[area.Area]; ok {
				perms = append(perms, domain.TemplateRolePermission{Area: area.Area, Access: acc})
			}
		}
		if _, err := s.templateSvc.CreateRole(ctx, templateID, spec.name, spec.description, perms); err != nil {
			return err
		}
	}
	return nil
}

func isAreaAvailable(availableFor []string, projectType string) bool {
	if len(availableFor) == 0 {
		return true
	}
	for _, v := range availableFor {
		if v == projectType {
			return true
		}
	}
	return false
}

// ensureInitialAdminUser создаёт первого администратора, если в системе
// нет ни одного активного пользователя с правом system.users.manage=full.
func (s *BootstrapService) ensureInitialAdminUser(ctx context.Context, role *domain.Role) error {
	count, err := s.roleRepo.CountActiveSystemAdmins(ctx)
	if err != nil {
		return errctx.Wrap(err, "ensureInitialAdminUser")
	}
	if count > 0 {
		return nil
	}

	password := s.cfg.InitialAdminPassword
	generated := false
	if password == "" {
		password, err = generateStrongPassword(16)
		if err != nil {
			return errctx.Wrap(err, "ensureInitialAdminUser")
		}
		generated = true
	}

	hash, err := s.passwordSvc.HashPassword(password)
	if err != nil {
		return errctx.Wrap(err, "ensureInitialAdminUser")
	}

	const (
		initialAdminFullName = "Фамилия Имя Отчество"
		initialAdminPosition = "Системный администратор"
	)

	user, err := s.userRepo.CreateUser(ctx,
		s.cfg.InitialAdminUsername,
		s.cfg.InitialAdminEmail,
		hash,
		initialAdminFullName,
		nil,
	)
	if err != nil {
		return errctx.Wrap(err, "ensureInitialAdminUser", "username", s.cfg.InitialAdminUsername)
	}

	position := initialAdminPosition
	if err := s.userRepo.UpdateProfile(ctx, user.ID,
		initialAdminFullName, s.cfg.InitialAdminEmail,
		&position, false, false, nil, nil); err != nil {
		return errctx.Wrap(err, "ensureInitialAdminUser", "userID", user.ID)
	}

	_ = s.userRepo.InsertPasswordHistory(ctx, user.ID, hash)

	if err := s.notificationSvc.InitializeDefaultSettings(ctx, user.ID.String()); err != nil {
		return errctx.Wrap(err, "ensureInitialAdminUser", "userID", user.ID)
	}

	if err := s.roleRepo.AssignRoleToUser(ctx, role.ID, user.ID); err != nil {
		return errctx.Wrap(err, "ensureInitialAdminUser", "userID", user.ID, "roleID", role.ID)
	}

	printInitialAdminCredentials(s.cfg.InitialAdminUsername, s.cfg.InitialAdminEmail, password, generated)
	log.Printf("[Bootstrap] first system administrator created: username=%s email=%s",
		s.cfg.InitialAdminUsername, s.cfg.InitialAdminEmail)

	return nil
}

// generateStrongPassword генерирует случайный пароль длины n (n >= 8),
// содержащий по меньшей мере одну заглавную, одну строчную латинскую
// букву, одну цифру и один спецсимвол.
func generateStrongPassword(n int) (string, error) {
	if n < 8 {
		n = 8
	}
	const (
		lower   = "abcdefghijkmnopqrstuvwxyz"
		upper   = "ABCDEFGHJKLMNPQRSTUVWXYZ"
		digits  = "23456789"
		special = "!@#$%^&*()-_=+[]{}"
	)
	all := lower + upper + digits + special

	required := []string{lower, upper, digits, special}
	out := make([]byte, 0, n)
	for _, set := range required {
		ch, err := randomByte(set)
		if err != nil {
			return "", err
		}
		out = append(out, ch)
	}
	for len(out) < n {
		ch, err := randomByte(all)
		if err != nil {
			return "", err
		}
		out = append(out, ch)
	}
	if err := shuffleBytes(out); err != nil {
		return "", err
	}
	return string(out), nil
}

func randomByte(alphabet string) (byte, error) {
	i, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
	if err != nil {
		return 0, err
	}
	return alphabet[i.Int64()], nil
}

func shuffleBytes(b []byte) error {
	for i := len(b) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		b[i], b[j.Int64()] = b[j.Int64()], b[i]
	}
	return nil
}

func printInitialAdminCredentials(username, email, password string, generated bool) {
	var passwordLine string
	if generated {
		passwordLine = "Пароль:  " + password
	} else {
		passwordLine = "Пароль:  задан администратором через переменную окружения INITIAL_ADMIN_PASSWORD"
	}

	banner := strings.Join([]string{
		"",
		"============================================================",
		"Projektus: создан первый системный администратор",
		"============================================================",
		fmt.Sprintf("Логин:   %s", username),
		fmt.Sprintf("E-mail:  %s", email),
		passwordLine,
		"",
	}, "\n")

	if generated {
		banner += strings.Join([]string{
			"ВНИМАНИЕ: Сохраните пароль немедленно — он выводится",
			"только один раз. После первого входа в систему",
			"рекомендуется сменить пароль через раздел «Профиль».",
		}, "\n") + "\n"
	}

	banner += "============================================================\n"

	fmt.Print(banner)
}
