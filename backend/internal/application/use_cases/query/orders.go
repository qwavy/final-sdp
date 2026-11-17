// Package query — use cases, которые ТОЛЬКО ЧИТАЮТ данные (CQS: запросы).
// Запросы ничего не меняют и собирают удобные для экрана «представления».
package query

import (
	"slices"
	"strings"

	"parcelhub/internal/application/port"
	"parcelhub/internal/domain"
)

type Option struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

// OrderView — заказ + имена связанных объектов + какие действия доступны.
type OrderView struct {
	domain.Order
	CustomerName    string   `json:"customerName"`
	CustomerPhone   string   `json:"customerPhone"`
	CourierName     string   `json:"courierName,omitempty"`
	PickupPointName string   `json:"pickupPointName,omitempty"`
	DistrictName    string   `json:"districtName"`
	DeliveryTitle   string   `json:"deliveryTitle"`
	StatusTitle     string   `json:"statusTitle"`
	NextStatuses    []Option `json:"nextStatuses"`
	Assignable      bool     `json:"assignable"`
	Editable        bool     `json:"editable"`
	Deletable       bool     `json:"deletable"`
}

type OrderQueries struct {
	orders       port.OrderRepository
	customers    port.CustomerRepository
	couriers     port.CourierRepository
	pickupPoints port.PickupPointRepository
}

func NewOrderQueries(
	orders port.OrderRepository,
	customers port.CustomerRepository,
	couriers port.CourierRepository,
	pickupPoints port.PickupPointRepository,
) *OrderQueries {
	return &OrderQueries{orders, customers, couriers, pickupPoints}
}

// List — заказы с фильтром по статусу и поиском, новые сверху.
func (q *OrderQueries) List(status, search string) ([]OrderView, error) {
	orders, err := q.orders.List()
	if err != nil {
		return nil, err
	}
	search = strings.ToLower(strings.TrimSpace(search))

	result := []OrderView{}
	for _, o := range orders {
		if status != "" && string(o.Status) != status {
			continue
		}
		v := q.view(o)
		text := strings.ToLower(v.ID + " " + v.Description + " " + v.CustomerName + " " + v.CustomerPhone)
		if search != "" && !strings.Contains(text, search) {
			continue
		}
		result = append(result, v)
	}
	slices.SortFunc(result, func(a, b OrderView) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return result, nil
}

func (q *OrderQueries) Get(id string) (OrderView, error) {
	order, err := q.orders.Get(id)
	if err != nil {
		return OrderView{}, err
	}
	return q.view(order), nil
}

// view подставляет имена вместо ID и считает доступные действия.
func (q *OrderQueries) view(o *domain.Order) OrderView {
	v := OrderView{
		Order:        *o,
		CustomerName: "(клиент удалён)",
		DistrictName: districtName(o.District),
		StatusTitle:  o.Status.Title(),
		NextStatuses: []Option{},
		Assignable:   o.CanAssign(),
		Editable:     o.CanEdit(),
		Deletable:    o.CanDelete(),
	}
	if c, err := q.customers.Get(o.CustomerID); err == nil {
		v.CustomerName, v.CustomerPhone = c.Name, c.Phone
	}
	if c, err := q.couriers.Get(o.CourierID); err == nil {
		v.CourierName = c.Name
	}
	if p, err := q.pickupPoints.Get(o.PickupPointID); err == nil {
		v.PickupPointName = p.Name + ", " + p.Address
	}
	if m, err := domain.NewDeliveryMethod(o.DeliveryType); err == nil {
		v.DeliveryTitle = m.Title()
	}
	for _, s := range o.NextStatuses() {
		if s != domain.StatusAssigned { // курьер назначается отдельной кнопкой
			v.NextStatuses = append(v.NextStatuses, Option{string(s), s.Title()})
		}
	}
	return v
}

// QuoteInput — данные формы для предварительного расчёта цены.
type QuoteInput struct {
	DeliveryType  string   `json:"deliveryType"`
	WeightKg      float64  `json:"weightKg"`
	DeclaredValue int64    `json:"declaredValue"`
	District      string   `json:"district"`
	PickupPointID string   `json:"pickupPointId"`
	Extras        []string `json:"extras"`
}

// QuoteOption — цена по одной стратегии.
type QuoteOption struct {
	Strategy   string             `json:"strategy"`
	Title      string             `json:"title"`
	DistanceKm float64            `json:"distanceKm"`
	Total      domain.Money       `json:"total"`
	Lines      []domain.PriceLine `json:"lines"`
}

// Quote считает цену сразу по всем стратегиям, ничего не сохраняя.
func (q *OrderQueries) Quote(in QuoteInput) ([]QuoteOption, error) {
	errs := domain.ValidationError{}
	district := in.District

	method, err := domain.NewDeliveryMethod(domain.DeliveryType(in.DeliveryType))
	if err != nil {
		errs["deliveryType"] = "выберите способ доставки"
	} else if method.NeedsPickupPoint() {
		if p, err := q.pickupPoints.Get(in.PickupPointID); err == nil {
			district = p.District
		} else {
			errs["pickupPointId"] = "выберите точку выдачи"
		}
	}
	if in.WeightKg <= 0 {
		errs["weightKg"] = "вес должен быть больше 0"
	}
	distance, ok := domain.DistanceFromHub(district)
	if !ok && errs["pickupPointId"] == "" {
		errs["district"] = "выберите район"
	}
	if err := errs.OrNil(); err != nil {
		return nil, err
	}

	result := []QuoteOption{}
	for _, s := range domain.PricingStrategies {
		price, err := domain.CalculatePrice(domain.PriceInput{
			Delivery:      method.Type(),
			Strategy:      s.Code(),
			WeightKg:      in.WeightKg,
			DistanceKm:    distance,
			DeclaredValue: domain.Money(in.DeclaredValue),
			Extras:        in.Extras,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, QuoteOption{s.Code(), s.Title(), distance, price.Total(), price.Lines()})
	}
	return result, nil
}

func districtName(code string) string {
	d, _ := domain.FindDistrict(code)
	return d.Name
}
