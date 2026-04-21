package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"projektus-backend/internal/domain"
)

// domainErrorBinding описывает, как доменная ошибка превращается в HTTP-ответ.
type domainErrorBinding struct {
	status  int
	code    string
	message string
}

// domainErrorTable — упорядоченная таблица соответствий. Первое совпадение
// через errors.Is возвращает связанный binding. Порядок важен: более
// специфичные ошибки идут раньше общих (ErrInvalidInput — последняя среди 400,
// потому что её часто используют как «общую» подложку).
var domainErrorTable = []struct {
	target  error
	binding domainErrorBinding
}{
	// 500 — инвариант БД нарушен
	{domain.ErrProjectAdminRoleMissing, domainErrorBinding{http.StatusInternalServerError, "PROJECT_ADMIN_ROLE_MISSING", "В проекте отсутствует роль администратора"}},

	// 404
	{domain.ErrNotFound, domainErrorBinding{http.StatusNotFound, "NOT_FOUND", "Ресурс не найден"}},

	// 403
	{domain.ErrAccessDenied, domainErrorBinding{http.StatusForbidden, "ACCESS_DENIED", "Нет доступа"}},
	{domain.ErrForbidden, domainErrorBinding{http.StatusForbidden, "FORBIDDEN", "Действие запрещено"}},

	// 409
	{domain.ErrActiveSprintExists, domainErrorBinding{http.StatusConflict, "ACTIVE_SPRINT_EXISTS", "В проекте уже есть активный спринт"}},
	{domain.ErrSprintDatesOverlap, domainErrorBinding{http.StatusConflict, "SPRINT_DATES_OVERLAP", "Даты спринта пересекаются с существующим"}},
	{domain.ErrAlreadyCancelled, domainErrorBinding{http.StatusConflict, "ALREADY_CANCELLED", "Уже отменено"}},
	{domain.ErrTagAlreadyExists, domainErrorBinding{http.StatusConflict, "TAG_ALREADY_EXISTS", "Тег уже существует"}},
	{domain.ErrConflict, domainErrorBinding{http.StatusConflict, "CONFLICT", "Конфликт состояния"}},

	// 400 — специфичные (по убыванию специфичности)
	{domain.ErrInvalidEstimation, domainErrorBinding{http.StatusBadRequest, "INVALID_ESTIMATION", "Оценка трудозатрат должна быть неотрицательным числом (до 2 знаков после точки)"}},
	{domain.ErrUserRequiresRole, domainErrorBinding{http.StatusBadRequest, "USER_REQUIRES_ROLE", "У пользователя должна быть назначена хотя бы одна системная роль"}},
	{domain.ErrMeetingInPast, domainErrorBinding{http.StatusBadRequest, "MEETING_IN_PAST", "Нельзя назначить встречу на прошедшее время"}},
	{domain.ErrInvalidTimeRange, domainErrorBinding{http.StatusBadRequest, "INVALID_TIME_RANGE", "Время окончания должно быть позже времени начала"}},
	{domain.ErrInvalidMeeting, domainErrorBinding{http.StatusBadRequest, "INVALID_MEETING", "Некорректные параметры встречи"}},
	{domain.ErrCannotRemoveOrganizer, domainErrorBinding{http.StatusBadRequest, "CANNOT_REMOVE_ORGANIZER", "Нельзя удалить организатора из списка участников"}},
	{domain.ErrColumnHasTasks, domainErrorBinding{http.StatusBadRequest, "COLUMN_HAS_TASKS", "В колонке есть задачи — удаление невозможно"}},
	{domain.ErrSwimlaneHasTasks, domainErrorBinding{http.StatusBadRequest, "SWIMLANE_HAS_TASKS", "В дорожке есть задачи — удаление невозможно"}},
	{domain.ErrRoleHasMembers, domainErrorBinding{http.StatusBadRequest, "ROLE_HAS_MEMBERS", "Роль назначена участникам — удаление невозможно"}},
	{domain.ErrProjectAdminRole, domainErrorBinding{http.StatusBadRequest, "PROJECT_ADMIN_ROLE", "Изменение роли администратора проекта запрещено"}},
	{domain.ErrTemplateAdminRole, domainErrorBinding{http.StatusBadRequest, "TEMPLATE_ADMIN_ROLE", "Изменение роли администратора шаблона запрещено"}},
	{domain.ErrSystemAdminRole, domainErrorBinding{http.StatusBadRequest, "SYSTEM_ADMIN_ROLE", "Системная роль администратора неизменяема"}},
	{domain.ErrLastProjectAdmin, domainErrorBinding{http.StatusBadRequest, "LAST_PROJECT_ADMIN", "Нельзя удалить последнего администратора проекта"}},
	{domain.ErrSystemParam, domainErrorBinding{http.StatusBadRequest, "SYSTEM_PARAM", "Системный параметр нельзя удалить"}},
	{domain.ErrSystemField, domainErrorBinding{http.StatusBadRequest, "SYSTEM_FIELD", "Системное поле нельзя изменять"}},
	{domain.ErrScrumWipNotAllowed, domainErrorBinding{http.StatusBadRequest, "SCRUM_WIP_NOT_ALLOWED", "WIP-лимиты на дорожках не поддерживаются в Scrum"}},
	{domain.ErrCompletedColumnWip, domainErrorBinding{http.StatusBadRequest, "COMPLETED_COLUMN_WIP", "Нельзя задать WIP-лимит для колонки «Выполнено»"}},
	{domain.ErrNoNextSprintForMove, domainErrorBinding{http.StatusBadRequest, "NO_NEXT_SPRINT", "Нет следующего запланированного спринта для переноса задач"}},
	{domain.ErrRequiredCustomFieldNotAllowed, domainErrorBinding{http.StatusBadRequest, "REQUIRED_CUSTOM_FIELD_NOT_ALLOWED", "Кастомные параметры не могут быть обязательными — обязательными могут быть только системные параметры"}},

	// 400 — общий случай (должен быть последним)
	{domain.ErrInvalidInput, domainErrorBinding{http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные параметры запроса"}},
}

// respondDomainErr пытается сопоставить err с доменной ошибкой из таблицы
// и записать соответствующий HTTP-ответ. Возвращает true, если ответ записан
// и вызывающему достаточно сделать `return`.
//
// Распаковывает *domain.ParamValidationError с его собственным пользовательским
// сообщением (через errors.As). Auth-ошибки (ErrInvalidCredentials,
// ErrUserBlocked, ErrIPBlocked, ErrTokenExpired и т.п.) в таблице
// намеренно отсутствуют — auth_handler обрабатывает их явно, учитывая
// специфику rate-limit и безопасных сообщений.
func respondDomainErr(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var pve *domain.ParamValidationError
	if errors.As(err, &pve) {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", pve.Message)
		return true
	}
	// InvalidPermissionCodeError несёт конкретные коды-нарушители — кладём их
	// в message, чтобы клиент видел, какие именно значения отклонены.
	var pce *domain.InvalidPermissionCodeError
	if errors.As(err, &pce) {
		writeError(c, http.StatusBadRequest, "INVALID_PERMISSION_CODE",
			"Неизвестные или неправильного scope коды прав: "+strings.Join(pce.Codes, ", "))
		return true
	}
	for _, item := range domainErrorTable {
		if errors.Is(err, item.target) {
			writeError(c, item.binding.status, item.binding.code, item.binding.message)
			return true
		}
	}
	return false
}

// respondInternal пишет 500 INTERNAL_ERROR и залогирует ошибку с контекстом
// метода/пути, чтобы в логах было видно, какой эндпоинт упал. Используется
// как «последняя линия обороны» после respondDomainErr.
//
// message — опциональный пользовательский текст; если пустой, берётся
// «Внутренняя ошибка сервера».
func respondInternal(c *gin.Context, err error, message string) {
	if message == "" {
		message = "Внутренняя ошибка сервера"
	}
	log.Printf("[%s %s] internal error: %v", c.Request.Method, c.FullPath(), err)
	writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}
