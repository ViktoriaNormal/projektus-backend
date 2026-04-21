package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// bindJSON инкапсулирует типичный сценарий:
//
//	var req dto.Xxx
//	if err := c.ShouldBindJSON(&req); err != nil {
//	    writeError(c, 400, "VALIDATION_ERROR", "Некорректные данные запроса")
//	    return
//	}
//
// в одну строку: `req, ok := bindJSON[dto.Xxx](c); if !ok { return }`.
//
// При ошибке пишет 400 VALIDATION_ERROR и возвращает ok=false. Текст сообщения
// унифицирован — фронт реагирует по полю `code`, а не по message.
func bindJSON[T any](c *gin.Context) (T, bool) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные данные запроса")
		return req, false
	}
	return req, true
}

// bindQuery — аналог bindJSON для query-параметров (используется в GET-эндпоинтах
// с form-тегами в DTO).
func bindQuery[T any](c *gin.Context) (T, bool) {
	var req T
	if err := c.ShouldBindQuery(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректные параметры запроса")
		return req, false
	}
	return req, true
}
