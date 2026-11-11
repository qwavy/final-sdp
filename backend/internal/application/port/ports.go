// Package port — интерфейсы, через которые приложение общается с внешним миром.
// Реализации лежат в adapters/out (хранилище, SMS). Благодаря этому логику
// можно тестировать и менять хранилище, не трогая use cases.
package port

import (
	"time"

	"parcelhub/internal/domain"
)

// Repository — хранилище сущностей одного типа.
type Repository[T any] interface {
	List() ([]T, error)
	Get(id string) (T, error)
	Save(item T) error
	Delete(id string) error
}

type (
	OrderRepository       = Repository[*domain.Order]
	CustomerRepository    = Repository[*domain.Customer]
	CourierRepository     = Repository[*domain.Courier]
	PickupPointRepository = Repository[*domain.PickupPoint]
)

// ActivityEntry — запись в журнале событий.
type ActivityEntry struct {
	ID      string    `json:"id"`
	At      time.Time `json:"at"`
	Kind    string    `json:"kind"`
	OrderID string    `json:"orderId,omitempty"`
	Message string    `json:"message"`
}

type ActivityRepository interface {
	Append(entry ActivityEntry) error
	List(limit int) ([]ActivityEntry, error)
}

// Notification — сообщение клиенту.
type Notification struct {
	Phone string
	Text  string
}

// Notifier — отправка уведомлений клиенту (реализация — SMS-адаптер).
type Notifier interface {
	Notify(n Notification) error
}

// ExportStore сохраняет выгруженные отчёты и возвращает путь к файлу.
type ExportStore interface {
	SaveExport(name string, data []byte) (string, error)
}

// EventPublisher — публикация доменных событий подписчикам.
type EventPublisher interface {
	Publish(events ...domain.Event)
}
