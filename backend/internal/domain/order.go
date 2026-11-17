package domain

import (
	"slices"
	"time"
)

type OrderStatus string

const (
	StatusCreated       OrderStatus = "created"
	StatusAssigned      OrderStatus = "assigned"
	StatusInTransit     OrderStatus = "in_transit"
	StatusAtPickupPoint OrderStatus = "at_pickup_point"
	StatusDelivered     OrderStatus = "delivered"
	StatusCancelled     OrderStatus = "cancelled"
)

// Statuses — все статусы по порядку.
var Statuses = []OrderStatus{
	StatusCreated, StatusAssigned, StatusInTransit, StatusAtPickupPoint, StatusDelivered, StatusCancelled,
}

var statusTitles = map[OrderStatus]string{
	StatusCreated:       "Создан",
	StatusAssigned:      "Курьер назначен",
	StatusInTransit:     "В пути",
	StatusAtPickupPoint: "Ждёт в точке выдачи",
	StatusDelivered:     "Выдан",
	StatusCancelled:     "Отменён",
}

func (s OrderStatus) Title() string { return statusTitles[s] }

type StatusChange struct {
	From OrderStatus `json:"from,omitempty"`
	To   OrderStatus `json:"to"`
	At   time.Time   `json:"at"`
}

// Order — заказ на доставку. Создаётся только через OrderBuilder.
type Order struct {
	ID              string         `json:"id"`
	CustomerID      string         `json:"customerId"`
	Description     string         `json:"description"`
	WeightKg        float64        `json:"weightKg"`
	DeclaredValue   Money          `json:"declaredValue"`
	DeliveryType    DeliveryType   `json:"deliveryType"`
	Address         string         `json:"address,omitempty"`
	PickupPointID   string         `json:"pickupPointId,omitempty"`
	District        string         `json:"district"`
	DistanceKm      float64        `json:"distanceKm"`
	PricingStrategy string         `json:"pricingStrategy"`
	Extras          []string       `json:"extras"`
	PriceLines      []PriceLine    `json:"priceLines"`
	Price           Money          `json:"price"`
	CourierID       string         `json:"courierId,omitempty"`
	Status          OrderStatus    `json:"status"`
	History         []StatusChange `json:"history"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// NextStatuses — куда можно перейти сейчас. Правила задаёт способ доставки (Factory Method).
func (o *Order) NextStatuses() []OrderStatus {
	method, err := NewDeliveryMethod(o.DeliveryType)
	if err != nil {
		return nil
	}
	return method.NextStatuses(o.Status)
}

func (o *Order) IsFinal() bool   { return o.Status == StatusDelivered || o.Status == StatusCancelled }
func (o *Order) CanEdit() bool   { return o.Status == StatusCreated }
func (o *Order) CanDelete() bool { return o.Status == StatusCreated || o.IsFinal() }
func (o *Order) CanAssign() bool { return slices.Contains(o.NextStatuses(), StatusAssigned) }

func (o *Order) AssignCourier(c *Courier, now time.Time) error {
	if !o.CanAssign() {
		return RuleViolation("назначить курьера можно только новому заказу")
	}
	if err := c.TakeOrder(); err != nil {
		return err
	}
	o.CourierID = c.ID
	o.setStatus(StatusAssigned, now)
	return nil
}

func (o *Order) ChangeStatus(to OrderStatus, now time.Time) error {
	if to == StatusAssigned {
		return RuleViolation("для этого статуса нажмите «Назначить курьера»")
	}
	if !slices.Contains(o.NextStatuses(), to) {
		return RuleViolation("нельзя перейти из «%s» в «%s»", o.Status.Title(), to.Title())
	}
	o.setStatus(to, now)
	return nil
}

func (o *Order) setStatus(to OrderStatus, now time.Time) {
	o.History = append(o.History, StatusChange{From: o.Status, To: to, At: now})
	o.Status = to
	o.UpdatedAt = now
}
