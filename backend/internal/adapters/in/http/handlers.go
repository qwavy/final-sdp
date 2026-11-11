package http

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"parcelhub/internal/application/use_cases/command"
	"parcelhub/internal/application/use_cases/query"
)

func (a *api) options(w http.ResponseWriter, _ *http.Request) error {
	writeJSON(w, http.StatusOK, a.facade.Options())
	return nil
}

// --- Заказы ---

func (a *api) listOrders(w http.ResponseWriter, r *http.Request) error {
	orders, err := a.facade.ListOrders(query.OrderFilter{
		Status: r.URL.Query().Get("status"),
		Search: r.URL.Query().Get("q"),
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, orders)
	return nil
}

func (a *api) getOrder(w http.ResponseWriter, r *http.Request) error {
	order, err := a.facade.GetOrder(r.PathValue("id"))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, order)
	return nil
}

func (a *api) quoteOrder(w http.ResponseWriter, r *http.Request) error {
	var in query.QuoteInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	options, err := a.facade.QuoteOrder(in)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, options)
	return nil
}

func (a *api) createOrder(w http.ResponseWriter, r *http.Request) error {
	var in command.OrderInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	order, err := a.facade.CreateOrder(in)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, order)
	return nil
}

func (a *api) updateOrder(w http.ResponseWriter, r *http.Request) error {
	var in command.OrderInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	order, err := a.facade.UpdateOrder(r.PathValue("id"), in)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, order)
	return nil
}

func (a *api) deleteOrder(w http.ResponseWriter, r *http.Request) error {
	if err := a.facade.DeleteOrder(r.PathValue("id")); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (a *api) assignCourier(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Selector string `json:"selector"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	order, err := a.facade.AssignCourier(r.PathValue("id"), in.Selector)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, order)
	return nil
}

func (a *api) changeStatus(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Status string `json:"status"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	order, err := a.facade.ChangeOrderStatus(r.PathValue("id"), in.Status)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, order)
	return nil
}

// --- Клиенты ---

func (a *api) listCustomers(w http.ResponseWriter, _ *http.Request) error {
	customers, err := a.facade.ListCustomers()
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, customers)
	return nil
}

func (a *api) createCustomer(w http.ResponseWriter, r *http.Request) error {
	var in command.CustomerInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	customer, err := a.facade.CreateCustomer(in)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, customer)
	return nil
}

func (a *api) updateCustomer(w http.ResponseWriter, r *http.Request) error {
	var in command.CustomerInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	customer, err := a.facade.UpdateCustomer(r.PathValue("id"), in)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, customer)
	return nil
}

func (a *api) deleteCustomer(w http.ResponseWriter, r *http.Request) error {
	if err := a.facade.DeleteCustomer(r.PathValue("id")); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// --- Курьеры ---

func (a *api) listCouriers(w http.ResponseWriter, _ *http.Request) error {
	couriers, err := a.facade.ListCouriers()
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, couriers)
	return nil
}

func (a *api) createCourier(w http.ResponseWriter, r *http.Request) error {
	var in command.CourierInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	courier, err := a.facade.CreateCourier(in)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, courier)
	return nil
}

func (a *api) updateCourier(w http.ResponseWriter, r *http.Request) error {
	var in command.CourierInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	courier, err := a.facade.UpdateCourier(r.PathValue("id"), in)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, courier)
	return nil
}

func (a *api) deleteCourier(w http.ResponseWriter, r *http.Request) error {
	if err := a.facade.DeleteCourier(r.PathValue("id")); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// --- Точки выдачи, журнал, отчёт ---

func (a *api) listPickupPoints(w http.ResponseWriter, _ *http.Request) error {
	points, err := a.facade.ListPickupPoints()
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, points)
	return nil
}

func (a *api) activity(w http.ResponseWriter, r *http.Request) error {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return badRequest{"limit должен быть положительным числом"}
		}
		limit = n
	}
	entries, err := a.facade.Activity(limit)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, entries)
	return nil
}

func (a *api) dailyReport(w http.ResponseWriter, r *http.Request) error {
	day, err := parseDay(r)
	if err != nil {
		return err
	}
	report, err := a.facade.DailyReport(day)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, report)
	return nil
}

func (a *api) exportDailyReport(w http.ResponseWriter, r *http.Request) error {
	day, err := parseDay(r)
	if err != nil {
		return err
	}
	file, err := a.facade.ExportDailyReport(day, r.URL.Query().Get("format"))
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", file.Name))
	w.Header().Set("X-Saved-To", file.SavedTo)
	_, err = w.Write(file.Data)
	return err
}

func parseDay(r *http.Request) (time.Time, error) {
	raw := r.URL.Query().Get("date")
	if raw == "" {
		return time.Now(), nil
	}
	day, err := time.ParseInLocation(time.DateOnly, raw, time.Local)
	if err != nil {
		return time.Time{}, badRequest{"дата в формате ГГГГ-ММ-ДД"}
	}
	return day, nil
}
