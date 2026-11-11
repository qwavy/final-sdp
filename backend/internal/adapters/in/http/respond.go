package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"parcelhub/internal/domain"
)

// apiFunc — обработчик, который возвращает ошибку вместо того, чтобы писать её сам.
type apiFunc func(w http.ResponseWriter, r *http.Request) error

func (h apiFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		writeError(w, err)
	}
}

type badRequest struct{ msg string }

func (e badRequest) Error() string { return e.msg }

var errNotFoundRoute = domain.NotFound("такого адреса API нет")

type errorBody struct {
	Error  string              `json:"error"`
	Fields []domain.FieldError `json:"fields,omitempty"`
}

// writeError выбирает HTTP-код по виду ошибки.
func writeError(w http.ResponseWriter, err error) {
	var validation *domain.ValidationError
	var bad badRequest
	switch {
	case errors.As(err, &validation):
		writeJSON(w, http.StatusUnprocessableEntity, errorBody{Error: "Проверьте введённые данные", Fields: validation.Fields})
	case errors.As(err, &bad):
		writeJSON(w, http.StatusBadRequest, errorBody{Error: bad.msg})
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorBody{Error: err.Error()})
	case errors.Is(err, domain.ErrInvalidTransition), errors.Is(err, domain.ErrRuleViolation):
		writeJSON(w, http.StatusConflict, errorBody{Error: err.Error()})
	default:
		slog.Error("внутренняя ошибка", "err", err)
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: "Внутренняя ошибка сервера"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("не удалось записать ответ", "err", err)
	}
}

const maxBodyBytes = 1 << 20

func readJSON(r *http.Request, dst any) error {
	body := http.MaxBytesReader(nil, r.Body, maxBodyBytes)
	if err := json.NewDecoder(body).Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return badRequest{"пустое тело запроса"}
		}
		return badRequest{"некорректный JSON: " + err.Error()}
	}
	return nil
}
