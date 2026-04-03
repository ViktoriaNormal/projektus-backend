package repositories

import (
	"context"

	"projektus-backend/internal/domain"
)

// =============================================================================
// Все справочные и системные данные по умолчанию.
// Единственное место, где определяются перечни — менять только здесь.
// =============================================================================

// --- Коды системных прав (используются в middleware и проверках) ---

const (
	SystemPermissionManageRoles          = "system.roles.manage"
	SystemPermissionManageUsers          = "system.users.manage"
	SystemPermissionManageProjects       = "system.projects.manage"
	SystemPermissionManagePasswordPolicy = "system.password_policy.manage"
	SystemPermissionManageTemplates      = "system.project_templates.manage"
)

// --- Права доступа ---

// AllPermissions — полный каталог прав доступа (системных и проектных).
var AllPermissions = []domain.PermissionDefinition{
	// Системные права
	{Code: "system.roles.manage", Scope: "system", Name: "Управление ролями", Description: "Создание, редактирование и удаление системных ролей"},
	{Code: "system.users.manage", Scope: "system", Name: "Управление пользователями", Description: "Создание, редактирование и удаление пользователей"},
	{Code: "system.projects.manage", Scope: "system", Name: "Управление всеми проектами", Description: "Просмотр, создание, редактирование, удаление всех проектов"},
	{Code: "system.password_policy.manage", Scope: "system", Name: "Управление парольной политикой", Description: "Настройка требований к паролям"},
	{Code: "system.project_templates.manage", Scope: "system", Name: "Управление шаблонами проектов", Description: "Создание, редактирование и удаление шаблонов проектов"},

	// Проектные права
	{Code: "project.boards", Scope: "project", Name: "Управление досками", Description: "Настройка досок, колонок и дорожек"},
	{Code: "project.tasks", Scope: "project", Name: "Управление задачами", Description: "Создание, редактирование и удаление задач"},
	{Code: "project.sprints", Scope: "project", Name: "Управление спринтами", Description: "Создание, редактирование и удаление спринтов"},
	{Code: "project.settings", Scope: "project", Name: "Настройки проекта", Description: "Управление настройками и параметрами проекта"},
	{Code: "project.members", Scope: "project", Name: "Управление участниками", Description: "Добавление и удаление участников проекта"},
	{Code: "project.roles", Scope: "project", Name: "Управление ролями проекта", Description: "Создание и настройка ролей проекта"},
	{Code: "project.analytics", Scope: "project", Name: "Аналитика и прогнозирование", Description: "Доступ к аналитике и отчётам проекта"},
}

// ProjectPermissionAreas — проектные области прав с привязкой к типу проекта (для UI назначения ролей).
var ProjectPermissionAreas = []domain.RefPermissionArea{
	{Area: "project.boards", Name: "Управление досками", Description: "Настройка досок, колонок и дорожек", AvailableFor: []string{"scrum", "kanban"}},
	{Area: "project.tasks", Name: "Управление задачами", Description: "Создание, редактирование и удаление задач", AvailableFor: []string{"scrum", "kanban"}},
	{Area: "project.sprints", Name: "Управление спринтами", Description: "Создание, редактирование и удаление спринтов", AvailableFor: []string{"scrum"}},
	{Area: "project.settings", Name: "Настройки проекта", Description: "Управление настройками и параметрами проекта", AvailableFor: []string{"scrum", "kanban"}},
	{Area: "project.members", Name: "Управление участниками", Description: "Добавление и удаление участников проекта", AvailableFor: []string{"scrum", "kanban"}},
	{Area: "project.roles", Name: "Управление ролями проекта", Description: "Создание и настройка ролей проекта", AvailableFor: []string{"scrum", "kanban"}},
	{Area: "project.analytics", Name: "Аналитика и прогнозирование", Description: "Доступ к аналитике и отчётам проекта", AvailableFor: []string{"scrum", "kanban"}},
}

// AccessLevels — уровни доступа для проектных прав.
var AccessLevels = []domain.RefKeyName{
	{Key: "none", Name: "Нет доступа"},
	{Key: "view", Name: "Просмотр"},
	{Key: "full", Name: "Полный доступ"},
}

// --- Типы колонок, полей, статусы ---

// ColumnSystemTypes — системные типы колонок доски.
var ColumnSystemTypes = []domain.RefColumnSystemType{
	{Key: "initial", Name: "Начальный", Description: "Задача создана, но не взята в работу"},
	{Key: "in_progress", Name: "В работе", Description: "Задача взята в работу"},
	{Key: "completed", Name: "Завершено", Description: "Задача выполнена"},
}

// FieldTypes — все допустимые типы параметров задач и проектов.
// AllowedScopes: "board_field" — поля задач доски, "project_param" — параметры проекта.
var FieldTypes = []domain.FieldTypeDefinition{
	{Key: "text", Name: "Текст", AvailableFor: []string{"scrum", "kanban"}, AllowedScopes: []string{"board_field", "project_param"}},
	{Key: "number", Name: "Число", AvailableFor: []string{"scrum", "kanban"}, AllowedScopes: []string{"board_field", "project_param"}},
	{Key: "datetime", Name: "Дата и время", AvailableFor: []string{"scrum", "kanban"}, AllowedScopes: []string{"board_field", "project_param"}},
	{Key: "select", Name: "Выбор", AvailableFor: []string{"scrum", "kanban"}, AllowedScopes: []string{"board_field", "project_param"}},
	{Key: "multiselect", Name: "Множественный выбор", AvailableFor: []string{"scrum", "kanban"}, AllowedScopes: []string{"board_field", "project_param"}},
	{Key: "checkbox", Name: "Чекбокс", AvailableFor: []string{"scrum", "kanban"}, AllowedScopes: []string{"board_field", "project_param"}},
	{Key: "user", Name: "Пользователь", AvailableFor: []string{"scrum", "kanban"}, AllowedScopes: []string{"board_field", "project_param"}},
	{Key: "user_list", Name: "Список пользователей", AvailableFor: []string{"scrum", "kanban"}, AllowedScopes: []string{"board_field", "project_param"}},
	{Key: "sprint", Name: "Спринт", AvailableFor: []string{"scrum"}, AllowedScopes: []string{"board_field"}},
	{Key: "sprint_list", Name: "Список спринтов", AvailableFor: []string{"scrum"}, AllowedScopes: []string{"board_field"}},
}

// ProjectStatuses — фиксированные статусы проекта.
var ProjectStatuses = []domain.RefKeyName{
	{Key: "active", Name: "Активный"},
	{Key: "archived", Name: "Архивирован"},
	{Key: "paused", Name: "Приостановлен"},
}

// --- Настройки досок ---

// EstimationUnits — единицы оценки трудозатрат.
var EstimationUnits = []domain.RefAvailable{
	{Key: "story_points", Name: "Story Points", AvailableFor: []string{"scrum"}},
	{Key: "time", Name: "Время", AvailableFor: []string{"scrum", "kanban"}},
}

// PriorityTypes — типы приоритизации с дефолтными значениями.
var PriorityTypes = []domain.RefPriorityType{
	{Key: "priority", Name: "Приоритет", AvailableFor: []string{"scrum", "kanban"}, DefaultValues: []string{"Низкий", "Средний", "Высокий", "Критичный"}},
	{Key: "service_class", Name: "Класс обслуживания", AvailableFor: []string{"kanban"}, DefaultValues: []string{"Ускоренный", "С фиксированной датой", "Стандартный", "Нематериальный"}},
}

// DefaultColumns — колонки по умолчанию для каждого типа проекта.
var DefaultColumns = map[string][]domain.DefaultColumnDef{
	"scrum": {
		{Name: "Бэклог спринта", SystemType: "initial", IsLocked: true},
		{Name: "В работе", SystemType: "in_progress", IsLocked: false},
		{Name: "На проверке", SystemType: "in_progress", IsLocked: false},
		{Name: "Выполнено", SystemType: "completed", IsLocked: false},
	},
	"kanban": {
		{Name: "Надо сделать", SystemType: "initial", IsLocked: false},
		{Name: "Готово к работе", SystemType: "initial", IsLocked: false},
		{Name: "В работе", SystemType: "in_progress", IsLocked: false},
		{Name: "На проверке", SystemType: "in_progress", IsLocked: false},
		{Name: "Выполнено", SystemType: "completed", IsLocked: false},
	},
}

// DefaultBoardFields — полный перечень системных полей доски.
// Единственное место, где определяется набор полей — сервисы берут данные отсюда.
var DefaultBoardFields = []domain.DefaultBoardFieldDef{
	{Key: "title", Name: "Название", FieldType: "text", IsRequired: true, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "description", Name: "Описание", FieldType: "text", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "status", Name: "Статус задачи", FieldType: "select", IsRequired: true, Options: []string{"Начальный", "В работе", "Завершено", "Отменено"}, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "author", Name: "Автор", FieldType: "user", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "assignee", Name: "Исполнитель", FieldType: "user", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "watchers", Name: "Наблюдатели", FieldType: "user_list", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "deadline", Name: "Дедлайн", FieldType: "datetime", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "priority", Name: "Приоритизация", FieldType: "priority", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "estimation", Name: "Оценка трудозатрат", FieldType: "estimation", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "sprint", Name: "Спринт", FieldType: "sprint", IsRequired: false, AvailableFor: []string{"scrum"}},
	{Key: "created_at", Name: "Дата создания", FieldType: "datetime", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
}

// =============================================================================
// ReferenceRepository — интерфейс и реализация (делегирует к переменным выше)
// =============================================================================

type ReferenceRepository interface {
	ListColumnSystemTypes(ctx context.Context) ([]domain.RefColumnSystemType, error)
	ListFieldTypes(ctx context.Context) ([]domain.FieldTypeDefinition, error)
	ListEstimationUnits(ctx context.Context) ([]domain.RefAvailable, error)
	ListPriorityTypes(ctx context.Context) ([]domain.RefPriorityType, error)
	ListPermissionAreas(ctx context.Context) ([]domain.RefPermissionArea, error)
	ListAccessLevels(ctx context.Context) ([]domain.RefKeyName, error)
}

type referenceRepository struct{}

func NewReferenceRepository() ReferenceRepository {
	return &referenceRepository{}
}

func (r *referenceRepository) ListColumnSystemTypes(_ context.Context) ([]domain.RefColumnSystemType, error) {
	return ColumnSystemTypes, nil
}

func (r *referenceRepository) ListFieldTypes(_ context.Context) ([]domain.FieldTypeDefinition, error) {
	return FieldTypes, nil
}

func (r *referenceRepository) ListEstimationUnits(_ context.Context) ([]domain.RefAvailable, error) {
	return EstimationUnits, nil
}

func (r *referenceRepository) ListPriorityTypes(_ context.Context) ([]domain.RefPriorityType, error) {
	return PriorityTypes, nil
}

func (r *referenceRepository) ListPermissionAreas(_ context.Context) ([]domain.RefPermissionArea, error) {
	return ProjectPermissionAreas, nil
}

func (r *referenceRepository) ListAccessLevels(_ context.Context) ([]domain.RefKeyName, error) {
	return AccessLevels, nil
}
