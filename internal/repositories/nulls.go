package repositories

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Этот файл — единственное место для конвертеров sql.Null*/uuid.NullUUID <->
// обычных Go-значений/указателей. Все репозитории используют именно эти
// функции, чтобы поведение «пустой строки = NULL», «nil = NULL», «NULL = nil»
// было одинаковым во всех местах.

// nullStringToPtr → *string (nil если Valid=false).
func nullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

// ptrToNullString: nil или "" → NULL. Непустая строка → Valid=true.
func ptrToNullString(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

// stringToNullString: "" → NULL, иначе Valid=true.
func stringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// nullStringOrEmpty: NULL → "", иначе хранимое значение. Полезен там, где
// domain-модель хранит поле как обычную строку без явного NULL.
func nullStringOrEmpty(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
}

// nullTimeToPtr: NULL → nil, иначе *time.Time.
func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	t := nt.Time
	return &t
}

// ptrToNullTime: nil → NULL.
func ptrToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// nullUUIDToPtr: NULL → nil, иначе *uuid.UUID.
func nullUUIDToPtr(n uuid.NullUUID) *uuid.UUID {
	if !n.Valid {
		return nil
	}
	id := n.UUID
	return &id
}

// ptrToNullUUID: nil → NULL.
func ptrToNullUUID(u *uuid.UUID) uuid.NullUUID {
	if u == nil {
		return uuid.NullUUID{Valid: false}
	}
	return uuid.NullUUID{UUID: *u, Valid: true}
}

// nullUUIDToStringPtr: NULL → nil, иначе *string со строковым UUID.
// Временный, нужен пока часть domain-моделей хранит ID как string.
// После унификации на uuid.UUID (фаза 5 плана) — удалить.
func nullUUIDToStringPtr(n uuid.NullUUID) *string {
	if !n.Valid {
		return nil
	}
	s := n.UUID.String()
	return &s
}

// nullUUIDToString: NULL → "", иначе строковый UUID. Временный, см. выше.
func nullUUIDToString(n uuid.NullUUID) string {
	if !n.Valid {
		return ""
	}
	return n.UUID.String()
}

// nullUUIDOrNil: NULL → uuid.Nil, иначе UUID. Используется там, где поле
// domain-модели не опциональное, но БД хранит NULL.
func nullUUIDOrNil(n uuid.NullUUID) uuid.UUID {
	if !n.Valid {
		return uuid.Nil
	}
	return n.UUID
}

// nullInt16ToPtr: NULL → nil, иначе *int16.
func nullInt16ToPtr(ni sql.NullInt16) *int16 {
	if !ni.Valid {
		return nil
	}
	v := ni.Int16
	return &v
}

// ptrToNullInt16: nil → NULL.
func ptrToNullInt16(v *int16) sql.NullInt16 {
	if v == nil {
		return sql.NullInt16{Valid: false}
	}
	return sql.NullInt16{Int16: *v, Valid: true}
}

// nullInt32ToPtr: NULL → nil, иначе *int32.
func nullInt32ToPtr(ni sql.NullInt32) *int32 {
	if !ni.Valid {
		return nil
	}
	v := ni.Int32
	return &v
}

// ptrToNullInt32: nil → NULL.
func ptrToNullInt32(v *int32) sql.NullInt32 {
	if v == nil {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: *v, Valid: true}
}

// mapSQLErr превращает sql.ErrNoRows в указанную доменную ошибку «не найдено».
// Остальные ошибки возвращаются как есть — вызывающий код сам решает,
// оборачивать ли их через errctx.Wrap.
//
// Использование:
//
//	row, err := r.q.GetFoo(ctx, id)
//	if err != nil {
//	    return nil, mapSQLErr(err, domain.ErrFooNotFound)
//	}
func mapSQLErr(err error, notFound error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return notFound
	}
	return err
}
