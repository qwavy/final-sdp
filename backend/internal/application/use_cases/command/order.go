// Package command — use cases, которые ИЗМЕНЯЮТ данные (CQS: команды).
// Команды возвращают только ошибку (и ID нового объекта), но не данные для показа.
// Данные для показа отдают запросы из пакета query.
package command

import (
	"errors"
	"fmt"
	"time"

	"parcelhub/internal/application/events"
	"parcelhub/internal/application/port"
	"parcelhub/internal/domain"
)

// OrderInput — данные формы заказа.
type OrderInput struct {
	CustomerID      string   `json:"customerId"`
	Description     string   `json:"description"`
	WeightKg        float64  `json:"weightKg"`
	DeclaredValue   int64    `json:"declaredValue"`
	DeliveryType    string   `json:"deliveryType"`
	Address         string   `json:"address"`
	District        string   `json:"district"`
	PickupPointID   string   `json:"pickupPointId"`
	PricingStrategy string   `json:"pricingStrategy"`
	Extras          []string `json:"extras"`
}

type OrderCommands struct {
	orders       port.OrderRepository
	customers    port.CustomerRepository
	couriers     port.CourierRepository
	pickupPoints port.PickupPointRepository
	publisher    *events.Publisher
	now          func() time.Time
}

func NewOrderCommands(
	orders port.OrderRepository,
	customers port.CustomerRepository,
	couriers port.CourierRepository,
	pickupPoints port.PickupPointRepository,
	publisher *events.Publisher,
	now func() time.Time,
) *OrderCommands {
	return &OrderCommands{orders, customers, couriers, pickupPoints, publisher, now}
}

func (c *OrderCommands) Create(in OrderInput) (string, error) {
	order, err := c.build(in)
	if err != nil {
		return "", err
	}
	if err := c.orders.Save(order); err != nil {
		return "", err
	}
	c.notify(events.OrderCreated, order.ID, fmt.Sprintf("Заказ %s создан", order.ID))
	return order.ID, nil
}

func (c *OrderCommands) Update(id string, in OrderInput) error {
	order, err := c.orders.Get(id)
	if err != nil {
		return err
	}
	if !order.CanEdit() {
		return domain.RuleViolation("изменять можно только заказ в статусе «%s»", domain.StatusCreated.Title())
	}
	edited, err := c.build(in)
	if err != nil {
		return err
	}
	if err := order.ApplyEdit(edited, c.now()); err != nil {
		return err
	}
	if err := c.orders.Save(order); err != nil {
		return err
	}
	c.notify(events.OrderUpdated, id, fmt.Sprintf("Заказ %s изменён", id))
	return nil
}

func (c *OrderCommands) Delete(id string) error {
	order, err := c.orders.Get(id)
	if err != nil {
		return err
	}
	if !order.CanDelete() {
		return domain.RuleViolation("удалить можно только новый или завершённый заказ — активный сначала отмените")
	}
	if err := c.orders.Delete(id); err != nil {
		return err
	}
	c.notify(events.OrderDeleted, id, fmt.Sprintf("Заказ %s удалён", id))
	return nil
}

// AssignCourier выбирает курьера стратегией, которую указал пользователь.
func (c *OrderCommands) AssignCourier(id, selectorCode string) error {
	order, err := c.orders.Get(id)
	if err != nil {
		return err
	}
	if !order.CanAssign() {
		return domain.InvalidTransition(order.Status, domain.StatusAssigned)
	}
	selector, err := domain.CourierSelectorByCode(selectorCode)
	if err != nil {
		return err
	}
	couriers, err := c.couriers.List()
	if err != nil {
		return err
	}
	courier, err := selector.Select(order, couriers)
	if err != nil {
		return err
	}
	from := order.Status
	if err := order.AssignCourier(courier, c.now()); err != nil {
		return err
	}
	if err := c.couriers.Save(courier); err != nil {
		return err
	}
	if err := c.orders.Save(order); err != nil {
		return err
	}
	c.notify(events.OrderStatusChanged, id, fmt.Sprintf("Заказ %s: «%s» → «%s» (%s)",
		id, from.Title(), order.Status.Title(), courier.Name))
	return nil
}

// ChangeStatus меняет статус. Когда заказ завершён, курьер освобождается.
func (c *OrderCommands) ChangeStatus(id, status string) error {
	order, err := c.orders.Get(id)
	if err != nil {
		return err
	}
	from := order.Status
	if err := order.ChangeStatus(domain.OrderStatus(status), c.now()); err != nil {
		return err
	}
	if order.IsFinal() && order.CourierID != "" {
		if err := c.releaseCourier(order.CourierID); err != nil {
			return err
		}
	}
	if err := c.orders.Save(order); err != nil {
		return err
	}
	c.notify(events.OrderStatusChanged, id, fmt.Sprintf("Заказ %s: «%s» → «%s»",
		id, from.Title(), order.Status.Title()))
	return nil
}

func (c *OrderCommands) releaseCourier(courierID string) error {
	courier, err := c.couriers.Get(courierID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil // курьера уже удалили — освобождать некого
	}
	if err != nil {
		return err
	}
	courier.ReleaseOrder()
	return c.couriers.Save(courier)
}

// build собирает заказ через Builder и проверяет ссылки на клиента и точку выдачи.
func (c *OrderCommands) build(in OrderInput) (*domain.Order, error) {
	b := domain.NewOrderBuilder().
		ForCustomer(in.CustomerID).
		Parcel(in.Description, in.WeightKg, domain.Money(in.DeclaredValue)).
		Delivery(domain.DeliveryType(in.DeliveryType)).
		ToAddress(in.Address, in.District).
		PricedBy(in.PricingStrategy).
		WithExtras(in.Extras...)

	if in.CustomerID != "" {
		if _, err := c.customers.Get(in.CustomerID); err != nil {
			b.Invalid("customerId", "клиент не найден")
		}
	}
	if in.PickupPointID != "" {
		point, err := c.pickupPoints.Get(in.PickupPointID)
		if err != nil {
			b.Invalid("pickupPointId", "точка выдачи не найдена")
		} else {
			b.ToPickupPoint(point)
		}
	}
	return b.Build(c.now())
}

// notify оповещает наблюдателей (паттерн Observer).
func (c *OrderCommands) notify(t events.Type, orderID, message string) {
	c.publisher.Notify(events.Event{Type: t, OrderID: orderID, Message: message, At: c.now()})
}
