package domain

import (
	"cmp"
	"slices"
	"strings"
)

// ПАТТЕРН STRATEGY (выбор курьера)
//
// Strategy         — CourierSelector.
// ConcreteStrategy — NearestCourierSelector, LeastLoadedCourierSelector.
// Context          — команда AssignCourier: пользователь выбирает стратегию кнопкой.

type CourierSelector interface {
	Code() string
	Title() string
	Select(order *Order, couriers []*Courier) (*Courier, error)
}

// NearestCourierSelector — ближайший к адресу заказа свободный курьер.
type NearestCourierSelector struct{}

func (NearestCourierSelector) Code() string  { return "nearest" }
func (NearestCourierSelector) Title() string { return "Ближайший курьер" }

func (NearestCourierSelector) Select(order *Order, couriers []*Courier) (*Courier, error) {
	free := freeCouriers(couriers)
	if len(free) == 0 {
		return nil, errNoFreeCourier()
	}
	return slices.MinFunc(free, func(a, b *Courier) int {
		return cmp.Or(
			cmp.Compare(distanceBetween(a.District, order.District), distanceBetween(b.District, order.District)),
			cmp.Compare(a.ActiveOrders, b.ActiveOrders),
			strings.Compare(a.Name, b.Name),
		)
	}), nil
}

// LeastLoadedCourierSelector — курьер с наименьшим числом активных заказов.
type LeastLoadedCourierSelector struct{}

func (LeastLoadedCourierSelector) Code() string  { return "least_loaded" }
func (LeastLoadedCourierSelector) Title() string { return "Наименее загруженный" }

func (LeastLoadedCourierSelector) Select(order *Order, couriers []*Courier) (*Courier, error) {
	free := freeCouriers(couriers)
	if len(free) == 0 {
		return nil, errNoFreeCourier()
	}
	return slices.MinFunc(free, func(a, b *Courier) int {
		return cmp.Or(
			cmp.Compare(a.ActiveOrders, b.ActiveOrders),
			cmp.Compare(distanceBetween(a.District, order.District), distanceBetween(b.District, order.District)),
			strings.Compare(a.Name, b.Name),
		)
	}), nil
}

var courierSelectors = []CourierSelector{
	NearestCourierSelector{},
	LeastLoadedCourierSelector{},
}

func CourierSelectors() []CourierSelector {
	return append([]CourierSelector(nil), courierSelectors...)
}

func CourierSelectorByCode(code string) (CourierSelector, error) {
	for _, s := range courierSelectors {
		if s.Code() == code {
			return s, nil
		}
	}
	return nil, RuleViolation("неизвестный способ выбора курьера %q", code)
}

func freeCouriers(couriers []*Courier) []*Courier {
	var free []*Courier
	for _, c := range couriers {
		if c.HasCapacity() {
			free = append(free, c)
		}
	}
	return free
}

func errNoFreeCourier() error {
	return RuleViolation("нет свободных курьеров: все заняты полностью")
}
