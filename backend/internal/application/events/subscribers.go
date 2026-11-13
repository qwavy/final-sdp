package events

import (
	"fmt"
	"log/slog"

	"parcelhub/internal/application/port"
	"parcelhub/internal/domain"
)

// ActivityRecorder записывает каждое событие в журнал.
type ActivityRecorder struct {
	activity port.ActivityRepository
}

func NewActivityRecorder(activity port.ActivityRepository) *ActivityRecorder {
	return &ActivityRecorder{activity: activity}
}

func (r *ActivityRecorder) Update(e Event) {
	err := r.activity.Append(port.ActivityEntry{
		At:      e.At,
		Kind:    string(e.Type),
		OrderID: e.OrderID,
		Message: e.Message,
	})
	if err != nil {
		slog.Error("не удалось записать событие в журнал", "err", err)
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

func (n *CustomerNotifier) Update(e Event) {
	if e.Type != OrderStatusChanged {
		return
	}
	order, err := n.orders.Get(e.OrderID)
	if err != nil {
		slog.Error("уведомление: заказ не найден", "order", e.OrderID, "err", err)
		return
	}
	customer, err := n.customers.Get(order.CustomerID)
	if err != nil {
		slog.Error("уведомление: клиент не найден", "customer", order.CustomerID, "err", err)
		return
	}

	text := n.message(order)
	entry := port.ActivityEntry{
		At:      e.At,
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
