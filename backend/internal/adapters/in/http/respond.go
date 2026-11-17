package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"parcelhub/internal/domain"
)

// handler — обработчик, который возвращает данные или ошибку.
// ServeHTTP превращает их в JSON-ответ, поэтому обработчики очень короткие.
type handler func(r *http.Request) (any, error)

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := h(r)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// badRequest — запрос нельзя разобрать (битый JSON, неверная дата).
type badRequest string

func (e badRequest) Error() string { return string(e) }

// writeError выбирает HTTP-код по виду ошибки.
func writeError(w http.ResponseWriter, err error) {
	var fields domain.ValidationError
	var bad badRequest
	switch {
	case errors.As(err, &fields):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": err.Error(), "fields": fields})
	case errors.As(err, &bad):
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
	case errors.Is(err, domain.ErrRule):
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
	default:
		slog.Error("внутренняя ошибка", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "внутренняя ошибка сервера"})
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func readJSON(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return badRequest("некорректный JSON")
	}
	return nil
}
