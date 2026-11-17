package http

import (
	"net/http"
	"strconv"
	"time"

	"parcelhub/internal/application/use_cases/command"
	"parcelhub/internal/application/use_cases/query"
)

func (a *api) options(r *http.Request) (any, error) {
	return a.facade.Options(), nil
}

// --- Заказы ---

func (a *api) listOrders(r *http.Request) (any, error) {
	return a.facade.ListOrders(r.URL.Query().Get("status"), r.URL.Query().Get("q"))
}

func (a *api) getOrder(r *http.Request) (any, error) {
	return a.facade.GetOrder(r.PathValue("id"))
}

func (a *api) quoteOrder(r *http.Request) (any, error) {
	var in query.QuoteInput
	if err := readJSON(r, &in); err != nil {
		return nil, err
	}
	return a.facade.QuoteOrder(in)
}

func (a *api) createOrder(r *http.Request) (any, error) {
	var in command.OrderInput
	if err := readJSON(r, &in); err != nil {
		return nil, err
	}
	return a.facade.CreateOrder(in)
}

func (a *api) updateOrder(r *http.Request) (any, error) {
	var in command.OrderInput
	if err := readJSON(r, &in); err != nil {
		return nil, err
	}
	return a.facade.UpdateOrder(r.PathValue("id"), in)
}

func (a *api) deleteOrder(r *http.Request) (any, error) {
	return nil, a.facade.DeleteOrder(r.PathValue("id"))
}

func (a *api) assignCourier(r *http.Request) (any, error) {
	var in struct {
		Selector string `json:"selector"`
	}
	if err := readJSON(r, &in); err != nil {
		return nil, err
	}
	return a.facade.AssignCourier(r.PathValue("id"), in.Selector)
}

func (a *api) changeStatus(r *http.Request) (any, error) {
	var in struct {
		Status string `json:"status"`
	}
	if err := readJSON(r, &in); err != nil {
		return nil, err
	}
	return a.facade.ChangeOrderStatus(r.PathValue("id"), in.Status)
}

// --- Клиенты ---

func (a *api) listCustomers(r *http.Request) (any, error) {
	return a.facade.ListCustomers()
}

func (a *api) createCustomer(r *http.Request) (any, error) {
	var in command.CustomerInput
	if err := readJSON(r, &in); err != nil {
		return nil, err
	}
	return a.facade.CreateCustomer(in)
}

func (a *api) updateCustomer(r *http.Request) (any, error) {
	var in command.CustomerInput
	if err := readJSON(r, &in); err != nil {
		return nil, err
	}
	return a.facade.UpdateCustomer(r.PathValue("id"), in)
}

func (a *api) deleteCustomer(r *http.Request) (any, error) {
	return nil, a.facade.DeleteCustomer(r.PathValue("id"))
}

// --- Курьеры ---

func (a *api) listCouriers(r *http.Request) (any, error) {
	return a.facade.ListCouriers()
}

func (a *api) createCourier(r *http.Request) (any, error) {
	var in command.CourierInput
	if err := readJSON(r, &in); err != nil {
		return nil, err
	}
	return a.facade.CreateCourier(in)
}

func (a *api) updateCourier(r *http.Request) (any, error) {
	var in command.CourierInput
	if err := readJSON(r, &in); err != nil {
		return nil, err
	}
	return a.facade.UpdateCourier(r.PathValue("id"), in)
}

func (a *api) deleteCourier(r *http.Request) (any, error) {
	return nil, a.facade.DeleteCourier(r.PathValue("id"))
}

// --- Точки выдачи, журнал, отчёт ---

func (a *api) listPickupPoints(r *http.Request) (any, error) {
	return a.facade.ListPickupPoints()
}

func (a *api) activity(r *http.Request) (any, error) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit")) // 0 — все записи
	return a.facade.Activity(limit)
}

func (a *api) dailyReport(r *http.Request) (any, error) {
	day, err := parseDay(r)
	if err != nil {
		return nil, err
	}
	return a.facade.DailyReport(day)
}

// exportDailyReport отдаёт файл, а не JSON, поэтому это обычный http-обработчик.
func (a *api) exportDailyReport(w http.ResponseWriter, r *http.Request) {
	day, err := parseDay(r)
	if err != nil {
		writeError(w, err)
		return
	}
	name, data, err := a.facade.ExportDailyReport(day, r.URL.Query().Get("format"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	_, _ = w.Write(data)
}

// parseDay читает ?date=ГГГГ-ММ-ДД, по умолчанию — сегодня.
func parseDay(r *http.Request) (time.Time, error) {
	raw := r.URL.Query().Get("date")
	if raw == "" {
		return time.Now(), nil
	}
	day, err := time.ParseInLocation(time.DateOnly, raw, time.Local)
	if err != nil {
		return time.Time{}, badRequest("дата в формате ГГГГ-ММ-ДД")
	}
	return day, nil
}
