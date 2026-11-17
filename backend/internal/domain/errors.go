package domain

import (
	"errors"
	"fmt"
)

// Два вида ошибок. По ним HTTP выбирает код ответа: 404 или 409.
var (
	ErrNotFound = errors.New("не найдено")
	ErrRule     = errors.New("нарушено правило")
)

// appError — ошибка с текстом для пользователя и видом (ErrNotFound или ErrRule).
type appError struct {
	kind error
	msg  string
}

func (e appError) Error() string { return e.msg }
func (e appError) Unwrap() error { return e.kind }

func NotFound(format string, args ...any) error {
	return appError{ErrNotFound, fmt.Sprintf(format, args...)}
}

func RuleViolation(format string, args ...any) error {
	return appError{ErrRule, fmt.Sprintf(format, args...)}
}

// ValidationError — ошибки полей формы: поле → текст ошибки.
type ValidationError map[string]string

func (e ValidationError) Error() string { return "Проверьте введённые данные" }

// OrNil возвращает nil, если ошибок нет.
func (e ValidationError) OrNil() error {
	if len(e) == 0 {
		return nil
	}
	return e
}
