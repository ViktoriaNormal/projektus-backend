// Package bootstrap собирает все зависимости приложения (repositories, services,
// handlers) в единый объект App. Цель — вынести длинный wire-up из main.go в
// тестируемую функцию и дать api.SetupRouter один параметр вместо 22.
package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	"projektus-backend/config"
	"projektus-backend/internal/api/handlers"
	"projektus-backend/internal/db"
	"projektus-backend/internal/repositories"
	"projektus-backend/internal/services"
)

// Repositories — набор всех репозиториев приложения. Собирается один раз
// в BuildRepositories.
type Repositories struct {
	User             repositories.UserRepository
	Auth             repositories.AuthRepository
	Notification     repositories.NotificationRepository
	Meeting          repositories.MeetingRepository
	Role             repositories.RoleRepository
	Project          repositories.ProjectRepository
	ProjectMember    repositories.ProjectMemberRepository
	ProjectRole      repositories.ProjectRoleRepository
	ProjectParam     repositories.ProjectParamRepository
	Template         repositories.TemplateRepository
	Board            repositories.BoardRepository
	Column           repositories.ColumnRepository
	Swimlane         repositories.SwimlaneRepository
	Note             repositories.NoteRepository
	BoardCustomField repositories.BoardCustomFieldRepository
	Task             repositories.TaskRepository
	Comment          repositories.CommentRepository
	Attachment       repositories.AttachmentRepository
	Checklist        repositories.ChecklistRepository
	TaskDependency   repositories.TaskDependencyRepository
	TaskWatcher      repositories.TaskWatcherRepository
	TaskFieldValue   repositories.TaskFieldValueRepository
	Sprint           repositories.SprintRepository
	SprintTask       repositories.SprintTaskRepository
	ProductBacklog   repositories.ProductBacklogRepository
	Tag              repositories.TagRepository
	AdminUser        repositories.AdminUserRepository
	PasswordPolicy   repositories.PasswordPolicyRepository
	Reference        repositories.ReferenceRepository
}

// BuildRepositories создаёт все репозитории от одного *db.Queries.
func BuildRepositories(q *db.Queries) *Repositories {
	return &Repositories{
		User:             repositories.NewUserRepository(q),
		Auth:             repositories.NewAuthRepository(q),
		Notification:     repositories.NewNotificationRepository(q),
		Meeting:          repositories.NewMeetingRepository(q),
		Role:             repositories.NewRoleRepository(q),
		Project:          repositories.NewProjectRepository(q),
		ProjectMember:    repositories.NewProjectMemberRepository(q),
		ProjectRole:      repositories.NewProjectRoleRepository(q),
		ProjectParam:    repositories.NewProjectParamRepository(q),
		Template:         repositories.NewTemplateRepository(q),
		Board:            repositories.NewBoardRepository(q),
		Column:           repositories.NewColumnRepository(q),
		Swimlane:         repositories.NewSwimlaneRepository(q),
		Note:             repositories.NewNoteRepository(q),
		BoardCustomField: repositories.NewBoardCustomFieldRepository(q),
		Task:             repositories.NewTaskRepository(q),
		Comment:          repositories.NewCommentRepository(q),
		Attachment:       repositories.NewAttachmentRepository(q),
		Checklist:        repositories.NewChecklistRepository(q),
		TaskDependency:   repositories.NewTaskDependencyRepository(q),
		TaskWatcher:      repositories.NewTaskWatcherRepository(q),
		TaskFieldValue:   repositories.NewTaskFieldValueRepository(q),
		Sprint:           repositories.NewSprintRepository(q),
		SprintTask:       repositories.NewSprintTaskRepository(q),
		ProductBacklog:   repositories.NewProductBacklogRepository(q),
		Tag:              repositories.NewTagRepository(q),
		AdminUser:        repositories.NewAdminUserRepository(q),
		PasswordPolicy:   repositories.NewPasswordPolicyRepository(q),
		Reference:        repositories.NewReferenceRepository(),
	}
}

// Services — набор прикладных сервисов. Некоторые зависят друг от друга,
// поэтому порядок создания важен (см. BuildServices).
type Services struct {
	Role             *services.RoleService
	Permission       *services.PermissionService
	Password         services.PasswordService
	PasswordPolicy   *services.PasswordPolicyService
	RateLimit        services.RateLimitService
	Notification     services.NotificationService
	Template         *services.TemplateService
	Auth             services.AuthService
	User             services.UserService
	Meeting          services.MeetingService
	ProjectRole      *services.ProjectRoleService
	ProjectParam     *services.ProjectParamService
	Tag              *services.TagService
	Board            *services.BoardService
	Task             *services.TaskService
	AdminUser        *services.AdminUserService
	Sprint           *services.SprintService
	ProductBacklog   *services.ProductBacklogService
	ProjectMember    *services.ProjectMemberService
	Project          *services.ProjectService
	ScrumAnalytics   *services.ScrumAnalyticsService
	KanbanAnalytics  *services.KanbanAnalyticsService
}

// BuildServices создаёт все сервисы от набора repositories + cfg + conn + queries.
func BuildServices(cfg *config.Config, repos *Repositories, conn *sql.DB, q *db.Queries) *Services {
	roleSvc := services.NewRoleService(repos.Role)
	permissionSvc := services.NewPermissionService(roleSvc, q)
	passwordSvc := services.NewPasswordService()
	passwordPolicySvc := services.NewPasswordPolicyService(repos.PasswordPolicy)
	rateLimitSvc := services.NewRateLimitService(cfg, repos.Auth)
	notificationSvc := services.NewNotificationService(repos.Notification)
	templateSvc := services.NewTemplateService(repos.Template, repos.Reference)
	authSvc := services.NewAuthService(cfg, repos.User, repos.Auth, passwordSvc, passwordPolicySvc, rateLimitSvc, roleSvc, notificationSvc)
	userSvc := services.NewUserService(repos.User)
	meetingSvc := services.NewMeetingService(repos.Meeting, notificationSvc)
	projectRoleSvc := services.NewProjectRoleService(repos.ProjectRole)
	projectParamSvc := services.NewProjectParamService(repos.ProjectParam, repos.User)
	tagSvc := services.NewTagService(repos.Tag)
	boardSvc := services.NewBoardService(repos.Board, repos.Column, repos.Swimlane, repos.Note, repos.BoardCustomField, repos.Task, conn)
	taskSvc := services.NewTaskService(repos.Task, repos.Project, repos.Tag, repos.Comment, repos.Attachment, repos.Checklist, repos.TaskDependency, repos.TaskWatcher, repos.TaskFieldValue, tagSvc, conn, q, notificationSvc)
	adminUserSvc := services.NewAdminUserService(repos.User, repos.AdminUser, roleSvc, passwordSvc, passwordPolicySvc, notificationSvc)
	sprintSvc := services.NewSprintService(repos.Sprint, repos.SprintTask, repos.ProductBacklog, repos.Task, repos.Column, repos.Project, tagSvc)
	productBacklogSvc := services.NewProductBacklogService(repos.ProductBacklog, repos.Task, repos.SprintTask)
	projectMemberSvc := services.NewProjectMemberService(repos.ProjectMember, repos.User, repos.Role, repos.ProjectRole)
	projectSvc := services.NewProjectService(repos.Project, templateSvc, repos.Board, repos.Column, repos.Swimlane, repos.Note, repos.BoardCustomField, repos.ProjectRole, repos.ProjectParam, repos.ProjectMember, conn)
	scrumAnalyticsSvc := services.NewScrumAnalyticsService(repos.Sprint, q, conn)
	kanbanAnalyticsSvc := services.NewKanbanAnalyticsService(q, conn)

	return &Services{
		Role:            roleSvc,
		Permission:      permissionSvc,
		Password:        passwordSvc,
		PasswordPolicy:  passwordPolicySvc,
		RateLimit:       rateLimitSvc,
		Notification:    notificationSvc,
		Template:        templateSvc,
		Auth:            authSvc,
		User:            userSvc,
		Meeting:         meetingSvc,
		ProjectRole:     projectRoleSvc,
		ProjectParam:    projectParamSvc,
		Tag:             tagSvc,
		Board:           boardSvc,
		Task:            taskSvc,
		AdminUser:       adminUserSvc,
		Sprint:          sprintSvc,
		ProductBacklog:  productBacklogSvc,
		ProjectMember:   projectMemberSvc,
		Project:         projectSvc,
		ScrumAnalytics:  scrumAnalyticsSvc,
		KanbanAnalytics: kanbanAnalyticsSvc,
	}
}

// Handlers — набор HTTP-хендлеров, создаваемых поверх services.
type Handlers struct {
	Auth               *handlers.AuthHandler
	User               *handlers.UserHandler
	Notification       *handlers.NotificationHandler
	Meeting            *handlers.MeetingHandler
	Role               *handlers.RoleHandler
	ProjectRole        *handlers.ProjectRoleHandler
	Tag                *handlers.TagHandler
	Task               *handlers.TaskHandler
	AdminUser          *handlers.AdminUserHandler
	AdminPasswordPolicy *handlers.AdminPasswordPolicyHandler
	ProductBacklog     *handlers.ProductBacklogHandler
	SprintBacklog      *handlers.SprintBacklogHandler
	ProjectMember      *handlers.ProjectMemberHandler
	Template           *handlers.TemplateHandler
	Project            *handlers.ProjectHandler
	Board              *handlers.BoardHandler
	ProjectParam       *handlers.ProjectParamHandler
	Sprint             *handlers.SprintHandler
	ScrumAnalytics     *handlers.ScrumAnalyticsHandler
	KanbanAnalytics    *handlers.KanbanAnalyticsHandler
}

// BuildHandlers создаёт все хендлеры от набора services + queries (там,
// где хендлер напрямую вызывает sqlc, как NotificationHandler).
func BuildHandlers(cfg *config.Config, svcs *Services, repos *Repositories, q *db.Queries) *Handlers {
	return &Handlers{
		Auth:                handlers.NewAuthHandler(cfg, svcs.Auth, svcs.Role),
		User:                handlers.NewUserHandler(svcs.User, repos.ProjectMember, repos.Role),
		Notification:        handlers.NewNotificationHandler(svcs.Notification, q),
		Meeting:             handlers.NewMeetingHandler(svcs.Meeting),
		Role:                handlers.NewRoleHandler(svcs.Role),
		ProjectRole:         handlers.NewProjectRoleHandler(svcs.ProjectRole, svcs.Permission),
		Tag:                 handlers.NewTagHandler(svcs.Tag),
		Task:                handlers.NewTaskHandler(svcs.Task, svcs.Board, svcs.Project, svcs.Permission),
		AdminUser:           handlers.NewAdminUserHandler(svcs.AdminUser),
		AdminPasswordPolicy: handlers.NewAdminPasswordPolicyHandler(svcs.PasswordPolicy),
		ProductBacklog:      handlers.NewProductBacklogHandler(svcs.ProductBacklog),
		SprintBacklog:       handlers.NewSprintBacklogHandler(svcs.Sprint),
		ProjectMember:       handlers.NewProjectMemberHandler(svcs.ProjectMember),
		Template:            handlers.NewTemplateHandler(svcs.Template),
		Project:             handlers.NewProjectHandler(svcs.Project, svcs.Template, svcs.Permission),
		Board:               handlers.NewBoardHandler(svcs.Board, svcs.Project, svcs.Permission),
		ProjectParam:        handlers.NewProjectParamHandler(svcs.ProjectParam, svcs.Project),
		Sprint:              handlers.NewSprintHandler(svcs.Sprint, svcs.Project),
		ScrumAnalytics:      handlers.NewScrumAnalyticsHandler(svcs.ScrumAnalytics),
		KanbanAnalytics:     handlers.NewKanbanAnalyticsHandler(svcs.KanbanAnalytics),
	}
}

// App — корневой объект приложения, созданный однократно в main.go.
// Предоставляет доступ к cfg, services и handlers для SetupRouter.
type App struct {
	Cfg      *config.Config
	Conn     *sql.DB
	Queries  *db.Queries
	Repos    *Repositories
	Services *Services
	Handlers *Handlers
}

// NewApp открывает подключение к БД, запускает bootstrap и собирает всё.
// Возвращает готовое приложение или ошибку. Вызывающий обязан закрыть
// app.Conn через defer.
func NewApp(cfg *config.Config) (*App, error) {
	conn, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := conn.PingContext(context.Background()); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	queries := db.New(conn)
	repos := BuildRepositories(queries)
	svcs := BuildServices(cfg, repos, conn, queries)

	// Первичная инициализация (идемпотентна): системные роли, пароль-политика,
	// шаблоны, первый администратор.
	bootstrapSvc := services.NewBootstrapService(cfg, repos.User, repos.Role, repos.PasswordPolicy, svcs.Template, svcs.Password, svcs.Notification)
	if err := bootstrapSvc.EnsureInitialState(context.Background()); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("bootstrap initial state: %w", err)
	}

	hs := BuildHandlers(cfg, svcs, repos, queries)
	return &App{
		Cfg:      cfg,
		Conn:     conn,
		Queries:  queries,
		Repos:    repos,
		Services: svcs,
		Handlers: hs,
	}, nil
}
