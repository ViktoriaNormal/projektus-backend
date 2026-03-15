// Package errctx предоставляет оборачивание ошибок с контекстом: место вызова (операция) и параметры.
package errctx

import (
	"fmt"
	"strings"
)

// Wrap оборачивает err, добавляя контекст операции op и пары key=value (args: key1, val1, key2, val2, ...).
// Если err == nil, возвращает nil. Цепочка ошибок сохраняется для errors.Is и errors.As.
func Wrap(err error, op string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", opWithArgs(op, args), err)
}

// WithOp создаёт новую ошибку с контекстом операции и параметров (без оборачивания другой ошибки).
func WithOp(msg string, op string, args ...interface{}) error {
	return fmt.Errorf("%s: %s", opWithArgs(op, args), msg)
}

func opWithArgs(op string, args []interface{}) string {
	if len(args) == 0 {
		return op
	}
	var b strings.Builder
	b.WriteString(op)
	b.WriteString("(")
	for i := 0; i < len(args); i += 2 {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprint(args[i]))
		if i+1 < len(args) {
			b.WriteString("=")
			b.WriteString(fmt.Sprint(args[i+1]))
		}
	}
	b.WriteString(")")
	return b.String()
}
