package domain

// ПАТТЕРН STRATEGY — выбор курьера.
//
// Strategy         — CourierSelector.
// ConcreteStrategy — NearestCourier, LeastLoadedCourier.
// Пользователь выбирает стратегию перед кнопкой «Назначить курьера».

type CourierSelector interface {
	Code() string
	Title() string
	Select(order *Order, couriers []*Courier) (*Courier, error)
}

// NearestCourier — ближайший к адресу заказа свободный курьер.
type NearestCourier struct{}

func (NearestCourier) Code() string  { return "nearest" }
func (NearestCourier) Title() string { return "Ближайший курьер" }

func (NearestCourier) Select(order *Order, couriers []*Courier) (*Courier, error) {
	var best *Courier
	for _, c := range couriers {
		if !c.HasCapacity() {
			continue
		}
		if best == nil || DistanceBetween(c.District, order.District) < DistanceBetween(best.District, order.District) {
			best = c
		}
	}
	if best == nil {
		return nil, RuleViolation("нет свободных курьеров")
	}
	return best, nil
}

// LeastLoadedCourier — свободный курьер с наименьшим числом заказов.
type LeastLoadedCourier struct{}

func (LeastLoadedCourier) Code() string  { return "least_loaded" }
func (LeastLoadedCourier) Title() string { return "Наименее загруженный" }

func (LeastLoadedCourier) Select(_ *Order, couriers []*Courier) (*Courier, error) {
	var best *Courier
	for _, c := range couriers {
		if !c.HasCapacity() {
			continue
		}
		if best == nil || c.ActiveOrders < best.ActiveOrders {
			best = c
		}
	}
	if best == nil {
		return nil, RuleViolation("нет свободных курьеров")
	}
	return best, nil
}

// CourierSelectors — все стратегии выбора курьера.
var CourierSelectors = []CourierSelector{NearestCourier{}, LeastLoadedCourier{}}

func CourierSelectorByCode(code string) (CourierSelector, error) {
	for _, s := range CourierSelectors {
		if s.Code() == code {
			return s, nil
		}
	}
	return nil, RuleViolation("неизвестный способ выбора курьера %q", code)
}
