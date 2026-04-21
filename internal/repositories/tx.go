package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"projektus-backend/internal/db"
)

// InTx оборачивает блок в BEGIN/COMMIT с автоматическим ROLLBACK при ошибке
// или панике. Возвращает ошибку коммита или ошибку из fn.
//
// Если fn возвращает err != nil, транзакция откатывается, и наружу возвращается
// та же err (без дополнительной обёртки, чтобы вызывающий мог делать
// errors.Is/As и получать domain-ошибки как обычно).
//
// Пример:
//
//	err := repositories.InTx(ctx, s.conn, func(qtx *db.Queries) error {
//	    if _, err := qtx.CreateX(ctx, params); err != nil {
//	        return err
//	    }
//	    return qtx.UpdateY(ctx, otherParams)
//	})
func InTx(ctx context.Context, conn *sql.DB, fn func(qtx *db.Queries) error) (err error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			err = fmt.Errorf("commit tx: %w", commitErr)
		}
	}()

	qtx := db.New(tx)
	return fn(qtx)
}

// InTxT — типизированный вариант InTx, возвращающий результат generic-типа T.
// Удобен, когда транзакционный блок должен вернуть результирующую сущность
// (например, созданную задачу вместе с вложенными списками).
func InTxT[T any](ctx context.Context, conn *sql.DB, fn func(qtx *db.Queries) (T, error)) (result T, err error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			err = fmt.Errorf("commit tx: %w", commitErr)
		}
	}()

	qtx := db.New(tx)
	result, err = fn(qtx)
	return
}
