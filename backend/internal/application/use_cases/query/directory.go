package query

import (
	"slices"
	"strings"

	"parcelhub/internal/application/port"
	"parcelhub/internal/domain"
)

// --- Справочники для форм ---

type DeliveryOption struct {
	Code             string       `json:"code"`
	Title            string       `json:"title"`
	BaseFee          domain.Money `json:"baseFee"`
	MaxWeightKg      float64      `json:"maxWeightKg"`
	NeedsPickupPoint bool         `json:"needsPickupPoint"`
}

type ExtraOption struct {
	Code  string `json:"code"`
	Title string `json:"title"`
	Hint  string `json:"hint"`
}

type Options struct {
	DeliveryTypes     []DeliveryOption `json:"deliveryTypes"`
	PricingStrategies []Option         `json:"pricingStrategies"`
	CourierSelectors  []Option         `json:"courierSelectors"`
	Extras            []ExtraOption    `json:"extras"`
	Districts         []Option         `json:"districts"`
	Statuses          []Option         `json:"statuses"`
}

// CatalogOptions — всё, из чего пользователь выбирает в формах.
func CatalogOptions() Options {
	var o Options
	for _, t := range domain.DeliveryTypes {
		m, _ := domain.NewDeliveryMethod(t)
		o.DeliveryTypes = append(o.DeliveryTypes, DeliveryOption{string(t), m.Title(), m.BaseFee(), m.MaxWeightKg(), m.NeedsPickupPoint()})
	}
	for _, s := range domain.PricingStrategies {
		o.PricingStrategies = append(o.PricingStrategies, Option{s.Code(), s.Title()})
	}
	for _, s := range domain.CourierSelectors {
		o.CourierSelectors = append(o.CourierSelectors, Option{s.Code(), s.Title()})
	}
	for _, e := range domain.Extras {
		o.Extras = append(o.Extras, ExtraOption{e.Code, e.Title, e.Hint})
	}
	for _, d := range domain.Districts {
		o.Districts = append(o.Districts, Option{d.Code, d.Name})
	}
	for _, s := range domain.Statuses {
		o.Statuses = append(o.Statuses, Option{string(s), s.Title()})
	}
	return o
}

// --- Клиенты, курьеры, точки выдачи, журнал ---

type CustomerView struct {
	domain.Customer
	TotalOrders  int `json:"totalOrders"`
	ActiveOrders int `json:"activeOrders"`
}

type CourierView struct {
	domain.Courier
	DistrictName string `json:"districtName"`
}

type PickupPointView struct {
	domain.PickupPoint
	DistrictName string `json:"districtName"`
	KindTitle    string `json:"kindTitle"`
}

type DirectoryQueries struct {
	customers    port.CustomerRepository
	couriers     port.CourierRepository
	pickupPoints port.PickupPointRepository
	orders       port.OrderRepository
	activity     port.ActivityRepository
}

func NewDirectoryQueries(
	customers port.CustomerRepository,
	couriers port.CourierRepository,
	pickupPoints port.PickupPointRepository,
	orders port.OrderRepository,
	activity port.ActivityRepository,
) *DirectoryQueries {
	return &DirectoryQueries{customers, couriers, pickupPoints, orders, activity}
}

func (q *DirectoryQueries) Customers() ([]CustomerView, error) {
	customers, err := q.customers.List()
	if err != nil {
		return nil, err
	}
	orders, err := q.orders.List()
	if err != nil {
		return nil, err
	}
	result := []CustomerView{}
	for _, c := range customers {
		v := CustomerView{Customer: *c}
		for _, o := range orders {
			if o.CustomerID == c.ID {
				v.TotalOrders++
				if !o.IsFinal() {
					v.ActiveOrders++
				}
			}
		}
		result = append(result, v)
	}
	slices.SortFunc(result, func(a, b CustomerView) int { return strings.Compare(a.Name, b.Name) })
	return result, nil
}

func (q *DirectoryQueries) Customer(id string) (CustomerView, error) {
	c, err := q.customers.Get(id)
	if err != nil {
		return CustomerView{}, err
	}
	return CustomerView{Customer: *c}, nil
}

func (q *DirectoryQueries) Couriers() ([]CourierView, error) {
	couriers, err := q.couriers.List()
	if err != nil {
		return nil, err
	}
	result := []CourierView{}
	for _, c := range couriers {
		result = append(result, CourierView{*c, districtName(c.District)})
	}
	slices.SortFunc(result, func(a, b CourierView) int { return strings.Compare(a.Name, b.Name) })
	return result, nil
}

func (q *DirectoryQueries) Courier(id string) (CourierView, error) {
	c, err := q.couriers.Get(id)
	if err != nil {
		return CourierView{}, err
	}
	return CourierView{*c, districtName(c.District)}, nil
}

func (q *DirectoryQueries) PickupPoints() ([]PickupPointView, error) {
	points, err := q.pickupPoints.List()
	if err != nil {
		return nil, err
	}
	result := []PickupPointView{}
	for _, p := range points {
		m, _ := domain.NewDeliveryMethod(p.Kind)
		result = append(result, PickupPointView{*p, districtName(p.District), m.Title()})
	}
	return result, nil
}

func (q *DirectoryQueries) Activity(limit int) ([]port.ActivityEntry, error) {
	return q.activity.List(limit)
}
