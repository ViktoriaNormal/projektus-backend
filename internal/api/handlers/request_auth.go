package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// requireUserUUID достаёт идентификатор текущего пользователя из контекста
// (положен туда в AuthMiddleware) и парсит в uuid.UUID. При ошибке сразу
// пишет 401 в ответ и возвращает ok=false — вызывающему остаётся сделать
// `return`.
//
// Используется во всех защищённых эндпоинтах вместо ручного
// `c.GetString("userID")` + `uuid.Parse` + `writeError(... UNAUTHORIZED ...)`.
func requireUserUUID(c *gin.Context) (uuid.UUID, bool) {
	raw := c.GetString("userID")
	if raw == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется аутентификация")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Не удалось определить пользователя")
		return uuid.Nil, false
	}
	return id, true
}

// paramUUID читает path-параметр c.Param(name) и парсит в uuid.UUID.
// При ошибке пишет 400 VALIDATION_ERROR с понятным текстом «Некорректный
// идентификатор <name>» и возвращает ok=false.
//
// Имя параметра передавать то же, что в маршруте (например, "projectId",
// "taskId").
func paramUUID(c *gin.Context, name string) (uuid.UUID, bool) {
	raw := c.Param(name)
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор "+name)
		return uuid.Nil, false
	}
	return id, true
}

// queryUUIDOpt читает опциональный query-параметр. Если он отсутствует —
// возвращает (nil, true): это нормальный сценарий. Если задан, но не парсится —
// пишет 400 и возвращает (nil, false).
func queryUUIDOpt(c *gin.Context, name string) (*uuid.UUID, bool) {
	raw := c.Query(name)
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный параметр "+name)
		return nil, false
	}
	return &id, true
}
