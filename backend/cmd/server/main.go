// Parcel Hub — сервер. Здесь собираются все слои (composition root):
// хранилище → наблюдатели (Observer) → фасад → HTTP.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	httpadapter "parcelhub/internal/adapters/in/http"
	"parcelhub/internal/adapters/out/notify"
	"parcelhub/internal/adapters/out/storage"
	"parcelhub/internal/application"
	"parcelhub/internal/application/events"
)

func main() {
	addr := flag.String("addr", ":8088", "адрес HTTP-сервера")
	dataDir := flag.String("data", "data", "папка с JSON-файлами")
	staticDir := flag.String("static", "../frontend/dist", "собранный фронтенд (необязательно)")
	flag.Parse()

	if err := run(*addr, *dataDir, *staticDir); err != nil {
		slog.Error("сервер остановлен с ошибкой", "err", err)
		os.Exit(1)
	}
}

func run(addr, dataDir, staticDir string) error {
	store, err := storage.Open(dataDir)
	if err != nil {
		return err
	}

	// Observer: подписываем наблюдателей на события заказов.
	publisher := events.NewPublisher()
	sms := notify.NewSmsNotifier(notify.NewSmsGatewayClient(filepath.Join(dataDir, "sms_outbox.jsonl")), "ParcelHub")
	publisher.Subscribe(events.NewActivityRecorder(store.Activity))
	publisher.Subscribe(events.NewCustomerNotifier(store.Orders, store.Customers, store.PickupPoints, sms, store.Activity))

	facade := application.NewDeliveryFacade(application.Deps{
		Orders:       store.Orders,
		Customers:    store.Customers,
		Couriers:     store.Couriers,
		PickupPoints: store.PickupPoints,
		Activity:     store.Activity,
		Exports:      store.Exports,
		Events:       publisher,
		Clock:        time.Now,
	})

	if store.IsEmpty() {
		slog.Info("первый запуск: загружаю тестовые данные")
		if err := seed(facade, store); err != nil {
			return err
		}
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           httpadapter.NewRouter(facade, staticDir),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	slog.Info("Parcel Hub запущен", "addr", addr, "data", dataDir)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
