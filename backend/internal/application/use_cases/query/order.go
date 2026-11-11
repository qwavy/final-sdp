// Package query — use cases, которые ТОЛЬКО ЧИТАЮТ данные (CQS: запросы).
// Запросы ничего не меняют и собирают удобные для интерфейса «представления».
package query

import (
	"cmp"
	"slices"
	"strings"

	"parcelhub/internal/application/port"
	"parcelhub/internal/domain"
)

type Option struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

// OrderView — заказ вместе с именами связанных объектов и доступными действиями.
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

type OrderFilter struct {
	Status string
	Search string
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

func (q *OrderQueries) List(f OrderFilter) ([]OrderView, error) {
	orders, err := q.orders.List()
	if err != nil {
		return nil, err
	}
	refs, err := loadReferences(q.customers, q.couriers, q.pickupPoints)
	if err != nil {
		return nil, err
	}
	search := strings.ToLower(strings.TrimSpace(f.Search))

	views := make([]OrderView, 0, len(orders))
	for _, o := range orders {
		if f.Status != "" && string(o.Status) != f.Status {
			continue
		}
		v := refs.orderView(o)
		if search != "" && !matches(v, search) {
			continue
		}
		views = append(views, v)
	}
	slices.SortFunc(views, func(a, b OrderView) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return views, nil
}

func (q *OrderQueries) Get(id string) (OrderView, error) {
	order, err := q.orders.Get(id)
	if err != nil {
		return OrderView{}, err
	}
	refs, err := loadReferences(q.customers, q.couriers, q.pickupPoints)
	if err != nil {
		return OrderView{}, err
	}
	return refs.orderView(order), nil
}

func matches(v OrderView, search string) bool {
	for _, field := range []string{v.ID, v.Description, v.CustomerName, v.CustomerPhone, v.CourierName} {
		if strings.Contains(strings.ToLower(field), search) {
			return true
		}
	}
	return false
}

// references — справочники для подстановки имён вместо ID.
type references struct {
	customers    map[string]*domain.Customer
	couriers     map[string]*domain.Courier
	pickupPoints map[string]*domain.PickupPoint
}

func loadReferences(
	customers port.CustomerRepository,
	couriers port.CourierRepository,
	pickupPoints port.PickupPointRepository,
) (references, error) {
	var refs references
	var err error
	if refs.customers, err = byID(customers, func(c *domain.Customer) string { return c.ID }); err != nil {
		return refs, err
	}
	if refs.couriers, err = byID(couriers, func(c *domain.Courier) string { return c.ID }); err != nil {
		return refs, err
	}
	refs.pickupPoints, err = byID(pickupPoints, func(p *domain.PickupPoint) string { return p.ID })
	return refs, err
}

func byID[T any](repo port.Repository[T], idOf func(T) string) (map[string]T, error) {
	items, err := repo.List()
	if err != nil {
		return nil, err
	}
	result := make(map[string]T, len(items))
	for _, item := range items {
		result[idOf(item)] = item
	}
	return result, nil
}

func (r references) orderView(o *domain.Order) OrderView {
	v := OrderView{
		Order:        *o,
		CustomerName: "(клиент удалён)",
		StatusTitle:  o.Status.Title(),
		Assignable:   o.CanAssign(),
		Editable:     o.CanEdit(),
		Deletable:    o.CanDelete(),
		NextStatuses: []Option{},
	}
	if c, ok := r.customers[o.CustomerID]; ok {
		v.CustomerName, v.CustomerPhone = c.Name, c.Phone
	}
	if c, ok := r.couriers[o.CourierID]; ok {
		v.CourierName = c.Name
	}
	if p, ok := r.pickupPoints[o.PickupPointID]; ok {
		v.PickupPointName = p.Name + ", " + p.Address
	}
	if d, ok := domain.FindDistrict(o.District); ok {
		v.DistrictName = d.Name
	}
	if m, err := domain.NewDeliveryMethod(o.DeliveryType); err == nil {
		v.DeliveryTitle = m.Title()
	}
	for _, s := range o.NextStatuses() {
		if s != domain.StatusAssigned { // курьер назначается отдельной кнопкой
			v.NextStatuses = append(v.NextStatuses, Option{Code: string(s), Title: s.Title()})
		}
	}
	return v
}

func compareByName(a, b string) int { return cmp.Compare(strings.ToLower(a), strings.ToLower(b)) }
