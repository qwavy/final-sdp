// Parcel Hub — сервер. Здесь собираются все слои:
// хранилище → наблюдатели (Observer) → фасад → HTTP.
package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"

	httpadapter "parcelhub/internal/adapters/in/http"
	"parcelhub/internal/adapters/out/notify"
	"parcelhub/internal/adapters/out/storage"
	"parcelhub/internal/application"
	"parcelhub/internal/application/events"
)

func main() {
	addr := flag.String("addr", ":8088", "адрес HTTP-сервера")
	dataDir := flag.String("data", "data", "папка с JSON-файлами")
	staticDir := flag.String("static", "../frontend/dist", "собранный фронтенд")
	flag.Parse()

	// 1. Хранилище (выходной адаптер)
	store, err := storage.Open(*dataDir)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Observer: подписываем наблюдателей
	sms := notify.NewSmsNotifier(&notify.SmsGateway{OutboxFile: filepath.Join(*dataDir, "sms_outbox.txt")})
	publisher := events.NewPublisher()
	publisher.Subscribe(events.NewActivityRecorder(store.Activity))
	publisher.Subscribe(events.NewCustomerNotifier(store.Orders, store.Customers, store.PickupPoints, sms, store.Activity))

	// 3. Фасад — вход в приложение
	facade := application.NewDeliveryFacade(application.Deps{
		Orders:       store.Orders,
		Customers:    store.Customers,
		Couriers:     store.Couriers,
		PickupPoints: store.PickupPoints,
		Activity:     store.Activity,
		Exports:      store.Exports,
		Publisher:    publisher,
	})

	// 4. Тестовые данные при первом запуске
	if store.IsEmpty() {
		log.Println("первый запуск: загружаю тестовые данные")
		if err := seed(facade, store); err != nil {
			log.Fatal(err)
		}
	}

	// 5. HTTP (входной адаптер)
	log.Printf("Parcel Hub запущен: http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, httpadapter.NewRouter(facade, *staticDir)))
}
