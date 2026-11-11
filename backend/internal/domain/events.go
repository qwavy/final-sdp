package domain

import "time"

// Event — доменное событие. На него подписываются наблюдатели (паттерн Observer).
type Event interface {
	EventName() string
	AggregateID() string
	OccurredAt() time.Time
}

type OrderCreated struct {
	OrderID string
	At      time.Time
}

type OrderUpdated struct {
	OrderID string
	At      time.Time
}

type OrderDeleted struct {
	OrderID string
	At      time.Time
}

type CourierAssigned struct {
	OrderID   string
	CourierID string
	At        time.Time
}

type OrderStatusChanged struct {
	OrderID string
	From    OrderStatus
	To      OrderStatus
	At      time.Time
}

func (e OrderCreated) EventName() string     { return "order.created" }
func (e OrderCreated) AggregateID() string   { return e.OrderID }
func (e OrderCreated) OccurredAt() time.Time { return e.At }

func (e OrderUpdated) EventName() string     { return "order.updated" }
func (e OrderUpdated) AggregateID() string   { return e.OrderID }
func (e OrderUpdated) OccurredAt() time.Time { return e.At }

func (e OrderDeleted) EventName() string     { return "order.deleted" }
func (e OrderDeleted) AggregateID() string   { return e.OrderID }
func (e OrderDeleted) OccurredAt() time.Time { return e.At }

func (e CourierAssigned) EventName() string     { return "order.courier_assigned" }
func (e CourierAssigned) AggregateID() string   { return e.OrderID }
func (e CourierAssigned) OccurredAt() time.Time { return e.At }

func (e OrderStatusChanged) EventName() string     { return "order.status_changed" }
func (e OrderStatusChanged) AggregateID() string   { return e.OrderID }
func (e OrderStatusChanged) OccurredAt() time.Time { return e.At }
