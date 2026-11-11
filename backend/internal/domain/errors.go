package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Виды ошибок. По ним HTTP-слой выбирает код ответа (404, 409 ...).
var (
	ErrNotFound          = errors.New("не найдено")
	ErrRuleViolation     = errors.New("нарушено правило")
	ErrInvalidTransition = errors.New("недопустимая смена статуса")
)

// DomainError — ошибка с понятным пользователю текстом и видом (ErrNotFound и т.д.).
type DomainError struct {
	kind error
	msg  string
}

func (e *DomainError) Error() string { return e.msg }
func (e *DomainError) Unwrap() error { return e.kind }

func NotFound(format string, args ...any) error {
	return &DomainError{kind: ErrNotFound, msg: fmt.Sprintf(format, args...)}
}

func RuleViolation(format string, args ...any) error {
	return &DomainError{kind: ErrRuleViolation, msg: fmt.Sprintf(format, args...)}
}

func InvalidTransition(from, to OrderStatus) error {
	return &DomainError{
		kind: ErrInvalidTransition,
		msg:  fmt.Sprintf("нельзя перейти из статуса «%s» в «%s»", from.Title(), to.Title()),
	}
}

// FieldError — ошибка конкретного поля формы.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError собирает все ошибки ввода, чтобы показать их пользователю сразу.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Add(field, message string) {
	e.Fields = append(e.Fields, FieldError{Field: field, Message: message})
}

func (e *ValidationError) Has(field string) bool {
	return slices.ContainsFunc(e.Fields, func(f FieldError) bool { return f.Field == field })
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		parts[i] = f.Field + ": " + f.Message
	}
	return "ошибка ввода: " + strings.Join(parts, "; ")
}

// OrNil возвращает nil, если ошибок нет.
func (e *ValidationError) OrNil() error {
	if len(e.Fields) == 0 {
		return nil
	}
	return e
}
