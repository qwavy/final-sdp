// Package events — ПАТТЕРН OBSERVER.
//
// Subject          — Publisher: хранит список наблюдателей и оповещает их.
// Observer         — интерфейс Observer с методом Update.
// ConcreteObserver — ActivityRecorder (журнал) и CustomerNotifier (SMS клиенту).
//
// Команда после изменения заказа вызывает publisher.Notify(event) и не знает,
// кто и как на это отреагирует. Новая реакция = новый Observer + Subscribe в main.go.
package events

import "time"

type Type string

const (
	OrderCreated       Type = "order.created"
	OrderUpdated       Type = "order.updated"
	OrderDeleted       Type = "order.deleted"
	OrderStatusChanged Type = "order.status_changed"
)

// Event — что произошло с заказом.
type Event struct {
	Type    Type
	OrderID string
	Message string
	At      time.Time
}

// Observer — тот, кто реагирует на события.
type Observer interface {
	Update(e Event)
}

// Publisher — Subject. Подписчики добавляются один раз при запуске (main.go).
type Publisher struct {
	observers []Observer
}

func NewPublisher() *Publisher { return &Publisher{} }

func (p *Publisher) Subscribe(o Observer) {
	p.observers = append(p.observers, o)
}

func (p *Publisher) Notify(e Event) {
	for _, o := range p.observers {
		o.Update(e)
	}
}
