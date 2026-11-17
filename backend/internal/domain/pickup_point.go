package domain

// PickupPoint — пункт выдачи или постамат (Kind).
// Точки создаются только тестовыми данными, поэтому проверок здесь нет.
type PickupPoint struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Address  string       `json:"address"`
	District string       `json:"district"`
	Kind     DeliveryType `json:"kind"`
}

func NewPickupPoint(name, address, district string, kind DeliveryType) *PickupPoint {
	return &PickupPoint{ID: NewID("PVZ"), Name: name, Address: address, District: district, Kind: kind}
}
