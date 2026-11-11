package query

import (
	"slices"

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
	for _, m := range domain.DeliveryMethods() {
		o.DeliveryTypes = append(o.DeliveryTypes, DeliveryOption{
			Code: string(m.Type()), Title: m.Title(), BaseFee: m.BaseFee(),
			MaxWeightKg: m.MaxWeightKg(), NeedsPickupPoint: m.NeedsPickupPoint(),
		})
	}
	for _, s := range domain.PricingStrategies() {
		o.PricingStrategies = append(o.PricingStrategies, Option{s.Code(), s.Title()})
	}
	for _, s := range domain.CourierSelectors() {
		o.CourierSelectors = append(o.CourierSelectors, Option{s.Code(), s.Title()})
	}
	for _, e := range domain.Extras() {
		o.Extras = append(o.Extras, ExtraOption{e.Code, e.Title, e.Hint})
	}
	for _, d := range domain.Districts() {
		o.Districts = append(o.Districts, Option{d.Code, d.Name})
	}
	for _, s := range domain.AllStatuses() {
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
	views := make([]CustomerView, 0, len(customers))
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
		views = append(views, v)
	}
	slices.SortFunc(views, func(a, b CustomerView) int { return compareByName(a.Name, b.Name) })
	return views, nil
}

func (q *DirectoryQueries) Customer(id string) (CustomerView, error) {
	views, err := q.Customers()
	if err != nil {
		return CustomerView{}, err
	}
	for _, v := range views {
		if v.ID == id {
			return v, nil
		}
	}
	return CustomerView{}, domain.NotFound("клиент %s не найден", id)
}

func (q *DirectoryQueries) Couriers() ([]CourierView, error) {
	couriers, err := q.couriers.List()
	if err != nil {
		return nil, err
	}
	views := make([]CourierView, 0, len(couriers))
	for _, c := range couriers {
		views = append(views, courierView(c))
	}
	slices.SortFunc(views, func(a, b CourierView) int { return compareByName(a.Name, b.Name) })
	return views, nil
}

func (q *DirectoryQueries) Courier(id string) (CourierView, error) {
	c, err := q.couriers.Get(id)
	if err != nil {
		return CourierView{}, err
	}
	return courierView(c), nil
}

func courierView(c *domain.Courier) CourierView {
	v := CourierView{Courier: *c}
	if d, ok := domain.FindDistrict(c.District); ok {
		v.DistrictName = d.Name
	}
	return v
}

func (q *DirectoryQueries) PickupPoints() ([]PickupPointView, error) {
	points, err := q.pickupPoints.List()
	if err != nil {
		return nil, err
	}
	views := make([]PickupPointView, 0, len(points))
	for _, p := range points {
		v := PickupPointView{PickupPoint: *p}
		if d, ok := domain.FindDistrict(p.District); ok {
			v.DistrictName = d.Name
		}
		if m, err := domain.NewDeliveryMethod(p.Kind); err == nil {
			v.KindTitle = m.Title()
		}
		views = append(views, v)
	}
	slices.SortFunc(views, func(a, b PickupPointView) int { return compareByName(a.Name, b.Name) })
	return views, nil
}

func (q *DirectoryQueries) Activity(limit int) ([]port.ActivityEntry, error) {
	return q.activity.List(limit)
}
