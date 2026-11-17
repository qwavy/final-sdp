package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// ПАТТЕРН BUILDER
//
// У заказа много полей, и часть зависит от способа доставки
// (адрес — для двери, точка выдачи — для ПВЗ и постамата).
// OrderBuilder собирает заказ по шагам, а Build() проверяет всё сразу
// и выдаёт готовый заказ — или ошибки по полям.
//
//	order, err := NewOrderBuilder().
//		ForCustomer("CUS-1").
//		Parcel("Книги", 2.5, 10000).
//		Delivery(DeliveryLocker).
//		ToPickupPoint(point).
//		PricedBy("by_weight").
//		WithExtras("fragile").
//		Build(time.Now())

type OrderBuilder struct {
	order       Order
	pickupPoint *PickupPoint
	errs        ValidationError
}

func NewOrderBuilder() *OrderBuilder {
	return &OrderBuilder{errs: ValidationError{}}
}

func (b *OrderBuilder) ForCustomer(customerID string) *OrderBuilder {
	b.order.CustomerID = customerID
	return b
}

func (b *OrderBuilder) Parcel(description string, weightKg float64, declaredValue Money) *OrderBuilder {
	b.order.Description = strings.TrimSpace(description)
	b.order.WeightKg = weightKg
	b.order.DeclaredValue = declaredValue
	return b
}

func (b *OrderBuilder) Delivery(t DeliveryType) *OrderBuilder {
	b.order.DeliveryType = t
	return b
}

// ToAddress — для доставки до двери.
func (b *OrderBuilder) ToAddress(address, district string) *OrderBuilder {
	b.order.Address = strings.TrimSpace(address)
	b.order.District = district
	return b
}

// ToPickupPoint — для ПВЗ или постамата.
func (b *OrderBuilder) ToPickupPoint(p *PickupPoint) *OrderBuilder {
	b.pickupPoint = p
	return b
}

func (b *OrderBuilder) PricedBy(strategy string) *OrderBuilder {
	b.order.PricingStrategy = strategy
	return b
}

func (b *OrderBuilder) WithExtras(codes ...string) *OrderBuilder {
	b.order.Extras = codes
	return b
}

// Invalid — ошибка, найденная снаружи (например, «клиент не найден»).
func (b *OrderBuilder) Invalid(field, message string) *OrderBuilder {
	b.errs[field] = message
	return b
}

// Build проверяет все поля, считает расстояние и цену.
func (b *OrderBuilder) Build(now time.Time) (*Order, error) {
	o := b.order
	errs := b.errs

	if o.CustomerID == "" && errs["customerId"] == "" {
		errs["customerId"] = "выберите клиента"
	}
	if o.Description == "" {
		errs["description"] = "опишите содержимое посылки"
	}
	if o.DeclaredValue < 0 {
		errs["declaredValue"] = "ценность не может быть меньше 0"
	}
	if slices.Contains(o.Extras, ExtraInsurance) && o.DeclaredValue <= 0 {
		errs["declaredValue"] = "для страховки укажите объявленную ценность"
	}
	if _, err := PricingStrategyByCode(o.PricingStrategy); err != nil {
		errs["pricingStrategy"] = "выберите способ расчёта цены"
	}

	method, err := NewDeliveryMethod(o.DeliveryType)
	if err != nil {
		errs["deliveryType"] = "выберите способ доставки"
	}
	if o.WeightKg <= 0 {
		errs["weightKg"] = "вес должен быть больше 0"
	} else if method != nil && o.WeightKg > method.MaxWeightKg() {
		errs["weightKg"] = fmt.Sprintf("для «%s» максимум %.0f кг", method.Title(), method.MaxWeightKg())
	}
	if method != nil {
		b.checkDestination(&o, method, errs)
	}

	if err := errs.OrNil(); err != nil {
		return nil, err
	}

	o.DistanceKm, _ = DistanceFromHub(o.District)
	price, err := CalculatePrice(PriceInput{
		Delivery:      o.DeliveryType,
		Strategy:      o.PricingStrategy,
		WeightKg:      o.WeightKg,
		DistanceKm:    o.DistanceKm,
		DeclaredValue: o.DeclaredValue,
		Extras:        o.Extras,
	})
	if err != nil {
		return nil, err
	}
	o.Price = price.Total()
	o.PriceLines = price.Lines()

	o.ID = NewID("ORD")
	o.Status = StatusCreated
	o.History = []StatusChange{{To: StatusCreated, At: now}}
	o.CreatedAt = now
	o.UpdatedAt = now
	return &o, nil
}

// checkDestination: для двери нужен адрес и район, для ПВЗ/постамата — подходящая точка.
func (b *OrderBuilder) checkDestination(o *Order, method DeliveryMethod, errs ValidationError) {
	if !method.NeedsPickupPoint() {
		o.PickupPointID = ""
		if o.Address == "" {
			errs["address"] = "укажите адрес"
		}
		if _, ok := FindDistrict(o.District); !ok {
			errs["district"] = "выберите район"
		}
		return
	}

	o.Address = ""
	p := b.pickupPoint
	switch {
	case p == nil:
		if errs["pickupPointId"] == "" {
			errs["pickupPointId"] = "выберите точку выдачи"
		}
	case p.Kind != method.Type():
		errs["pickupPointId"] = fmt.Sprintf("«%s» не подходит для способа «%s»", p.Name, method.Title())
	default:
		o.PickupPointID = p.ID
		o.District = p.District
	}
}
