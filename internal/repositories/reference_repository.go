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
	{Code: "system.projects.manage", Scope: "system", Name: "Управление проектами", Description: "Просмотр, создание, редактирование, удаление, архивация и разархивация всех проектов"},
	{Code: "system.password_policy.manage", Scope: "system", Name: "Управление парольной политикой", Description: "Настройка требований к паролям"},
	{Code: "system.project_templates.manage", Scope: "system", Name: "Управление шаблонами проектов", Description: "Создание, редактирование и удаление шаблонов проектов"},

	// Проектные права
	{Code: "project.boards.manage", Scope: "project", Name: "Управление досками", Description: "Настройка досок, колонок и дорожек"},
	{Code: "project.tasks.manage", Scope: "project", Name: "Управление задачами", Description: "Создание, редактирование и удаление задач"},
	{Code: "project.project_settings.manage", Scope: "project", Name: "Настройки проекта", Description: "Управление настройками и параметрами проекта"},
	{Code: "project.sprints.manage", Scope: "project", Name: "Управление спринтами", Description: "Создание, редактирование и удаление спринтов"},
	{Code: "project.backlog.manage", Scope: "project", Name: "Управление бэклогом", Description: "Управление продуктовым бэклогом"},
	{Code: "project.analytics.manage", Scope: "project", Name: "Аналитика", Description: "Доступ к аналитике и отчётам проекта"},
	{Code: "project.wip_limits.manage", Scope: "project", Name: "WIP-лимиты", Description: "Настройка лимитов незавершённой работы"},
}

// ProjectPermissionAreas — проектные области прав с привязкой к типу проекта (для UI назначения ролей).
var ProjectPermissionAreas = []domain.RefPermissionArea{
	{Area: "project.boards.manage", Name: "Доски", Description: "Управление досками проекта", AvailableFor: []string{"scrum", "kanban"}},
	{Area: "project.tasks.manage", Name: "Задачи", Description: "Управление задачами", AvailableFor: []string{"scrum", "kanban"}},
	{Area: "project.project_settings.manage", Name: "Настройки проекта", Description: "Управление настройками проекта", AvailableFor: []string{"scrum", "kanban"}},
	{Area: "project.sprints.manage", Name: "Спринты", Description: "Управление спринтами", AvailableFor: []string{"scrum"}},
	{Area: "project.backlog.manage", Name: "Бэклог", Description: "Управление бэклогом продукта", AvailableFor: []string{"scrum"}},
	{Area: "project.analytics.manage", Name: "Аналитика", Description: "Доступ к аналитике и отчётам", AvailableFor: []string{"scrum", "kanban"}},
	{Area: "project.wip_limits.manage", Name: "WIP-лимиты", Description: "Настройка лимитов незавершённой работы", AvailableFor: []string{"kanban"}},
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

// --- Системные параметры шаблонов (для справочника) ---

// SystemTaskFields — системные параметры задач (нередактируемые в шаблонах).
var SystemTaskFields = []domain.RefSystemField{
	{Key: "priority", Name: "Приоритет", FieldType: "select", AvailableFor: []string{"scrum", "kanban"}, Description: "Приоритет задачи"},
	{Key: "estimation", Name: "Оценка", FieldType: "number", AvailableFor: []string{"scrum", "kanban"}, Description: "Оценка трудозатрат"},
}

// SystemProjectParams — системные параметры проекта (нередактируемые в шаблонах).
var SystemProjectParams = []domain.RefSystemProjectParam{
	{Key: "sprint_duration", Name: "Длительность спринта", FieldType: "number", IsRequired: false, Options: nil},
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
	{Key: "title", Name: "Название", Description: "Название задачи", FieldType: "text", IsRequired: true, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "description", Name: "Описание", Description: "Подробное описание задачи", FieldType: "text", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "status", Name: "Статус задачи", Description: "Текущий статус задачи", FieldType: "select", IsRequired: true, Options: []string{"Начальный", "В работе", "Завершено", "Отменено"}, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "author", Name: "Автор", Description: "Создатель задачи", FieldType: "user", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "assignee", Name: "Исполнитель", Description: "Ответственный за выполнение", FieldType: "user", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "watchers", Name: "Наблюдатели", Description: "Пользователи, следящие за задачей", FieldType: "user_list", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "deadline", Name: "Дедлайн", Description: "Крайний срок выполнения", FieldType: "datetime", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "priority", Name: "Приоритизация", Description: "Приоритет задачи", FieldType: "priority", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "estimation", Name: "Оценка трудозатрат", Description: "Оценка объёма работы", FieldType: "estimation", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
	{Key: "sprint", Name: "Спринт", Description: "Спринт, к которому относится задача", FieldType: "sprint", IsRequired: false, AvailableFor: []string{"scrum"}},
	{Key: "created_at", Name: "Дата создания", Description: "Дата и время создания задачи", FieldType: "datetime", IsRequired: false, AvailableFor: []string{"scrum", "kanban"}},
}

// PriorityDescriptions — описания поля «Приоритизация» в зависимости от priority_type.
var PriorityDescriptions = map[string]string{
	"priority":      "Приоритет задачи",
	"service_class": "Класс обслуживания задачи",
}

// EstimationDescriptions — описания поля «Оценка трудозатрат» в зависимости от estimation_unit.
var EstimationDescriptions = map[string]string{
	"story_points": "Оценка объёма работы в Story Points",
	"time":         "Оценка объёма работы в формате времени",
}

// =============================================================================
// ReferenceRepository — интерфейс и реализация (делегирует к переменным выше)
// =============================================================================

type ReferenceRepository interface {
	ListColumnSystemTypes(ctx context.Context) ([]domain.RefColumnSystemType, error)
	ListFieldTypes(ctx context.Context) ([]domain.FieldTypeDefinition, error)
	ListEstimationUnits(ctx context.Context) ([]domain.RefAvailable, error)
	ListPriorityTypes(ctx context.Context) ([]domain.RefPriorityType, error)
	ListSystemTaskFields(ctx context.Context) ([]domain.RefSystemField, error)
	ListSystemProjectParams(ctx context.Context) ([]domain.RefSystemProjectParam, error)
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

func (r *referenceRepository) ListSystemTaskFields(_ context.Context) ([]domain.RefSystemField, error) {
	return SystemTaskFields, nil
}

func (r *referenceRepository) ListSystemProjectParams(_ context.Context) ([]domain.RefSystemProjectParam, error) {
	return SystemProjectParams, nil
}

func (r *referenceRepository) ListPermissionAreas(_ context.Context) ([]domain.RefPermissionArea, error) {
	return ProjectPermissionAreas, nil
}

func (r *referenceRepository) ListAccessLevels(_ context.Context) ([]domain.RefKeyName, error) {
	return AccessLevels, nil
}
