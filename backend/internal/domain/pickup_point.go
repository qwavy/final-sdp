package domain

import "strings"

// PickupPoint — пункт выдачи заказов или постамат (Kind).
type PickupPoint struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Address  string       `json:"address"`
	District string       `json:"district"`
	Kind     DeliveryType `json:"kind"`
}

func NewPickupPoint(name, address, district string, kind DeliveryType) (*PickupPoint, error) {
	p := &PickupPoint{
		ID:       NewID("PVZ"),
		Name:     strings.TrimSpace(name),
		Address:  strings.TrimSpace(address),
		District: district,
		Kind:     kind,
	}
	var errs ValidationError
	if p.Name == "" {
		errs.Add("name", "укажите название")
	}
	if p.Address == "" {
		errs.Add("address", "укажите адрес")
	}
	if _, ok := FindDistrict(district); !ok {
		errs.Add("district", "выберите район")
	}
	if kind != DeliveryPickupPoint && kind != DeliveryLocker {
		errs.Add("kind", "тип точки: пункт выдачи или постамат")
	}
	if err := errs.OrNil(); err != nil {
		return nil, err
	}
	return p, nil
}
