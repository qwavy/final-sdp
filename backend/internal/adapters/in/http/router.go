// Package http — входной адаптер: REST API.
// Обработчики только читают запрос и вызывают DeliveryFacade.
package http

import (
	"net/http"

	"parcelhub/internal/application"
)

type api struct {
	facade *application.DeliveryFacade
}

// NewRouter — все маршруты API. staticDir — собранный фронтенд.
func NewRouter(facade *application.DeliveryFacade, staticDir string) http.Handler {
	a := &api{facade: facade}
	mux := http.NewServeMux()

	mux.Handle("GET /api/options", handler(a.options))

	mux.Handle("GET /api/orders", handler(a.listOrders))
	mux.Handle("POST /api/orders", handler(a.createOrder))
	mux.Handle("POST /api/orders/quote", handler(a.quoteOrder))
	mux.Handle("GET /api/orders/{id}", handler(a.getOrder))
	mux.Handle("PUT /api/orders/{id}", handler(a.updateOrder))
	mux.Handle("DELETE /api/orders/{id}", handler(a.deleteOrder))
	mux.Handle("POST /api/orders/{id}/assign", handler(a.assignCourier))
	mux.Handle("POST /api/orders/{id}/status", handler(a.changeStatus))

	mux.Handle("GET /api/customers", handler(a.listCustomers))
	mux.Handle("POST /api/customers", handler(a.createCustomer))
	mux.Handle("PUT /api/customers/{id}", handler(a.updateCustomer))
	mux.Handle("DELETE /api/customers/{id}", handler(a.deleteCustomer))

	mux.Handle("GET /api/couriers", handler(a.listCouriers))
	mux.Handle("POST /api/couriers", handler(a.createCourier))
	mux.Handle("PUT /api/couriers/{id}", handler(a.updateCourier))
	mux.Handle("DELETE /api/couriers/{id}", handler(a.deleteCourier))

	mux.Handle("GET /api/pickup-points", handler(a.listPickupPoints))
	mux.Handle("GET /api/activity", handler(a.activity))
	mux.Handle("GET /api/reports/daily", handler(a.dailyReport))
	mux.HandleFunc("GET /api/reports/daily/export", a.exportDailyReport)

	mux.Handle("/", http.FileServer(http.Dir(staticDir)))
	return mux
}
