// Package http — входной адаптер: REST API для браузера.
// Обработчики только разбирают запрос и вызывают DeliveryFacade.
package http

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"parcelhub/internal/application"
)

type api struct {
	facade *application.DeliveryFacade
}

// NewRouter собирает все маршруты. staticDir — собранный фронтенд (если есть).
func NewRouter(facade *application.DeliveryFacade, staticDir string) http.Handler {
	a := &api{facade: facade}
	mux := http.NewServeMux()

	mux.Handle("GET /api/options", apiFunc(a.options))

	mux.Handle("GET /api/orders", apiFunc(a.listOrders))
	mux.Handle("POST /api/orders", apiFunc(a.createOrder))
	mux.Handle("POST /api/orders/quote", apiFunc(a.quoteOrder))
	mux.Handle("GET /api/orders/{id}", apiFunc(a.getOrder))
	mux.Handle("PUT /api/orders/{id}", apiFunc(a.updateOrder))
	mux.Handle("DELETE /api/orders/{id}", apiFunc(a.deleteOrder))
	mux.Handle("POST /api/orders/{id}/assign", apiFunc(a.assignCourier))
	mux.Handle("POST /api/orders/{id}/status", apiFunc(a.changeStatus))

	mux.Handle("GET /api/customers", apiFunc(a.listCustomers))
	mux.Handle("POST /api/customers", apiFunc(a.createCustomer))
	mux.Handle("PUT /api/customers/{id}", apiFunc(a.updateCustomer))
	mux.Handle("DELETE /api/customers/{id}", apiFunc(a.deleteCustomer))

	mux.Handle("GET /api/couriers", apiFunc(a.listCouriers))
	mux.Handle("POST /api/couriers", apiFunc(a.createCourier))
	mux.Handle("PUT /api/couriers/{id}", apiFunc(a.updateCourier))
	mux.Handle("DELETE /api/couriers/{id}", apiFunc(a.deleteCourier))

	mux.Handle("GET /api/pickup-points", apiFunc(a.listPickupPoints))
	mux.Handle("GET /api/activity", apiFunc(a.activity))
	mux.Handle("GET /api/reports/daily", apiFunc(a.dailyReport))
	mux.Handle("GET /api/reports/daily/export", apiFunc(a.exportDailyReport))

	mux.Handle("/api/", apiFunc(func(http.ResponseWriter, *http.Request) error {
		return errNotFoundRoute
	}))

	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		mux.Handle("/", spaHandler(staticDir))
	}
	return logRequests(mux)
}

// spaHandler отдаёт файлы фронтенда, а на неизвестные пути — index.html.
func spaHandler(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		files.ServeHTTP(w, r)
	})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			slog.Info("http", "method", r.Method, "path", r.URL.Path, "took", time.Since(start).Round(time.Microsecond))
		}
	})
}
