package events

import (
	"fmt"
	"log/slog"

	"parcelhub/internal/application/port"
	"parcelhub/internal/domain"
)

// Describe — понятный человеку текст события.
func Describe(e domain.Event) string {
	switch ev := e.(type) {
	case domain.OrderCreated:
		return fmt.Sprintf("Заказ %s создан", ev.OrderID)
	case domain.OrderUpdated:
		return fmt.Sprintf("Заказ %s изменён", ev.OrderID)
	case domain.OrderDeleted:
		return fmt.Sprintf("Заказ %s удалён", ev.OrderID)
	case domain.CourierAssigned:
		return fmt.Sprintf("На заказ %s назначен курьер %s", ev.OrderID, ev.CourierID)
	case domain.OrderStatusChanged:
		return fmt.Sprintf("Заказ %s: «%s» → «%s»", ev.OrderID, ev.From.Title(), ev.To.Title())
	default:
		return e.EventName()
	}
}

// ActivityRecorder записывает каждое событие в журнал.
type ActivityRecorder struct {
	activity port.ActivityRepository
}

func NewActivityRecorder(activity port.ActivityRepository) *ActivityRecorder {
	return &ActivityRecorder{activity: activity}
}

func (r *ActivityRecorder) Handle(e domain.Event) {
	err := r.activity.Append(port.ActivityEntry{
		At:      e.OccurredAt(),
		Kind:    e.EventName(),
		OrderID: e.AggregateID(),
		Message: Describe(e),
	})
	if err != nil {
		slog.Error("не удалось записать событие в журнал", "event", e.EventName(), "err", err)
	}
}

// CustomerNotifier отправляет клиенту SMS, когда меняется статус его заказа.
type CustomerNotifier struct {
	orders       port.OrderRepository
	customers    port.CustomerRepository
	pickupPoints port.PickupPointRepository
	notifier     port.Notifier
	activity     port.ActivityRepository
}

func NewCustomerNotifier(
	orders port.OrderRepository,
	customers port.CustomerRepository,
	pickupPoints port.PickupPointRepository,
	notifier port.Notifier,
	activity port.ActivityRepository,
) *CustomerNotifier {
	return &CustomerNotifier{orders, customers, pickupPoints, notifier, activity}
}

func (n *CustomerNotifier) Handle(e domain.Event) {
	changed, ok := e.(domain.OrderStatusChanged)
	if !ok {
		return
	}
	order, err := n.orders.Get(changed.OrderID)
	if err != nil {
		slog.Error("уведомление: заказ не найден", "order", changed.OrderID, "err", err)
		return
	}
	customer, err := n.customers.Get(order.CustomerID)
	if err != nil {
		slog.Error("уведомление: клиент не найден", "customer", order.CustomerID, "err", err)
		return
	}

	text := n.message(order)
	entry := port.ActivityEntry{
		At:      changed.At,
		Kind:    "sms.sent",
		OrderID: order.ID,
		Message: fmt.Sprintf("SMS на %s: %s", customer.Phone, text),
	}
	if err := n.notifier.Notify(port.Notification{Phone: customer.Phone, Text: text}); err != nil {
		entry.Kind = "sms.failed"
		entry.Message = fmt.Sprintf("SMS на %s не отправлено: %v", customer.Phone, err)
	}
	if err := n.activity.Append(entry); err != nil {
		slog.Error("не удалось записать SMS в журнал", "err", err)
	}
}

func (n *CustomerNotifier) message(order *domain.Order) string {
	text := fmt.Sprintf("Заказ %s: %s.", order.ID, order.Status.Title())
	if order.Status == domain.StatusAtPickupPoint {
		if point, err := n.pickupPoints.Get(order.PickupPointID); err == nil {
			text += fmt.Sprintf(" Заберите: %s, %s.", point.Name, point.Address)
		}
	}
	return text
}
