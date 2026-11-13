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

var statusTitles = map[OrderStatus]string{
	StatusCreated:       "Создан",
	StatusAssigned:      "Курьер назначен",
	StatusInTransit:     "В пути",
	StatusAtPickupPoint: "Ждёт в точке выдачи",
	StatusDelivered:     "Выдан",
	StatusCancelled:     "Отменён",
}

var allStatuses = []OrderStatus{
	StatusCreated, StatusAssigned, StatusInTransit, StatusAtPickupPoint, StatusDelivered, StatusCancelled,
}

func AllStatuses() []OrderStatus { return slices.Clone(allStatuses) }

func (s OrderStatus) Title() string {
	if t, ok := statusTitles[s]; ok {
		return t
	}
	return string(s)
}

func (s OrderStatus) Valid() bool { _, ok := statusTitles[s]; return ok }

type StatusChange struct {
	From OrderStatus `json:"from,omitempty"`
	To   OrderStatus `json:"to"`
	At   time.Time   `json:"at"`
}

// Order — заказ на доставку посылки. Создаётся только через OrderBuilder.
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

// NextStatuses — куда можно перевести заказ сейчас (правила задаёт способ доставки).
func (o *Order) NextStatuses() []OrderStatus {
	method, err := NewDeliveryMethod(o.DeliveryType)
	if err != nil {
		return nil
	}
	return method.NextStatuses(o.Status)
}

func (o *Order) canMoveTo(to OrderStatus) bool { return slices.Contains(o.NextStatuses(), to) }

func (o *Order) IsFinal() bool   { return o.Status == StatusDelivered || o.Status == StatusCancelled }
func (o *Order) CanEdit() bool   { return o.Status == StatusCreated }
func (o *Order) CanDelete() bool { return o.Status == StatusCreated || o.IsFinal() }
func (o *Order) CanAssign() bool { return o.canMoveTo(StatusAssigned) }

func (o *Order) AssignCourier(c *Courier, now time.Time) error {
	if !o.CanAssign() {
		return InvalidTransition(o.Status, StatusAssigned)
	}
	if err := c.TakeOrder(); err != nil {
		return err
	}
	o.CourierID = c.ID
	o.moveTo(StatusAssigned, now)
	return nil
}

func (o *Order) ChangeStatus(to OrderStatus, now time.Time) error {
	if !to.Valid() {
		return RuleViolation("неизвестный статус %q", to)
	}
	if to == StatusAssigned {
		return RuleViolation("статус «%s» ставится через «Назначить курьера»", to.Title())
	}
	if !o.canMoveTo(to) {
		return InvalidTransition(o.Status, to)
	}
	o.moveTo(to, now)
	return nil
}

// ApplyEdit переносит изменения из заново собранного заказа (только пока он «Создан»).
func (o *Order) ApplyEdit(edited *Order, now time.Time) error {
	if !o.CanEdit() {
		return RuleViolation("изменять можно только заказ в статусе «%s»", StatusCreated.Title())
	}
	o.CustomerID = edited.CustomerID
	o.Description = edited.Description
	o.WeightKg = edited.WeightKg
	o.DeclaredValue = edited.DeclaredValue
	o.DeliveryType = edited.DeliveryType
	o.Address = edited.Address
	o.PickupPointID = edited.PickupPointID
	o.District = edited.District
	o.DistanceKm = edited.DistanceKm
	o.PricingStrategy = edited.PricingStrategy
	o.Extras = edited.Extras
	o.PriceLines = edited.PriceLines
	o.Price = edited.Price
	o.UpdatedAt = now
	return nil
}

func (o *Order) moveTo(to OrderStatus, now time.Time) {
	from := o.Status
	o.Status = to
	o.UpdatedAt = now
	o.History = append(o.History, StatusChange{From: from, To: to, At: now})
}
