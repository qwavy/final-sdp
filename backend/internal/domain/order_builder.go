package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// ПАТТЕРН BUILDER
//
// Builder — OrderBuilder. У заказа много полей, часть из них зависит от способа
// доставки (адрес или пункт выдачи). Builder собирает заказ по шагам, а Build()
// проверяет всё сразу, считает расстояние и цену и выдаёт готовый заказ
// в корректном состоянии — или список ошибок по полям.
//
//	order, err := NewOrderBuilder().
//		ForCustomer("CUS-1").
//		Parcel("Книги", 2.5, 10000).
//		Delivery(DeliveryLocker).
//		ToPickupPoint(point).
//		PricedBy("by_weight").
//		WithExtras(ExtraFragile).
//		Build(time.Now())

const MaxDeclaredValue Money = 5_000_000

type OrderBuilder struct {
	order       Order
	pickupPoint *PickupPoint
	errs        ValidationError
}

func NewOrderBuilder() *OrderBuilder { return &OrderBuilder{} }

func (b *OrderBuilder) ForCustomer(customerID string) *OrderBuilder {
	b.order.CustomerID = strings.TrimSpace(customerID)
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

// ToAddress — адрес для доставки до двери.
func (b *OrderBuilder) ToAddress(address, districtCode string) *OrderBuilder {
	b.order.Address = strings.TrimSpace(address)
	b.order.District = districtCode
	return b
}

// ToPickupPoint — пункт выдачи или постамат.
func (b *OrderBuilder) ToPickupPoint(p *PickupPoint) *OrderBuilder {
	b.pickupPoint = p
	return b
}

func (b *OrderBuilder) PricedBy(strategyCode string) *OrderBuilder {
	b.order.PricingStrategy = strategyCode
	return b
}

func (b *OrderBuilder) WithExtras(codes ...string) *OrderBuilder {
	b.order.Extras = codes
	return b
}

// Invalid добавляет ошибку, найденную снаружи (например, «клиент не найден»).
func (b *OrderBuilder) Invalid(field, message string) *OrderBuilder {
	b.errs.Add(field, message)
	return b
}

func (b *OrderBuilder) Build(now time.Time) (*Order, error) {
	o := b.order
	errs := ValidationError{Fields: slices.Clone(b.errs.Fields)}

	if o.CustomerID == "" && !errs.Has("customerId") {
		errs.Add("customerId", "выберите клиента")
	}
	switch {
	case o.Description == "":
		errs.Add("description", "опишите содержимое посылки")
	case utf8.RuneCountInString(o.Description) > 200:
		errs.Add("description", "описание не длиннее 200 символов")
	}
	if o.DeclaredValue < 0 || o.DeclaredValue > MaxDeclaredValue {
		errs.Add("declaredValue", fmt.Sprintf("ценность от 0 до %d ₸", MaxDeclaredValue))
	}

	method, err := NewDeliveryMethod(o.DeliveryType)
	if err != nil {
		errs.Add("deliveryType", "выберите способ доставки")
	}
	switch {
	case o.WeightKg <= 0:
		errs.Add("weightKg", "вес должен быть больше 0")
	case method != nil && o.WeightKg > method.MaxWeightKg():
		errs.Add("weightKg", fmt.Sprintf("для способа «%s» максимум %.0f кг", method.Title(), method.MaxWeightKg()))
	}
	if method != nil {
		b.resolveDestination(&o, method, &errs)
	}
	if _, err := PricingStrategyByCode(o.PricingStrategy); err != nil {
		errs.Add("pricingStrategy", "выберите способ расчёта цены")
	}
	o.Extras = validateExtras(o.Extras, o.DeclaredValue, &errs)

	if err := errs.OrNil(); err != nil {
		return nil, err
	}

	o.DistanceKm, _ = DistanceFromHub(o.District)
	price, err := BuildPrice(QuoteSpec{
		DeliveryType:  o.DeliveryType,
		PricingCode:   o.PricingStrategy,
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
	o.CreatedAt, o.UpdatedAt = now, now
	return &o, nil
}

// resolveDestination: для ПВЗ/постамата берём точку выдачи, для курьера — адрес и район.
func (b *OrderBuilder) resolveDestination(o *Order, method DeliveryMethod, errs *ValidationError) {
	if !method.NeedsPickupPoint() {
		o.PickupPointID = ""
		if o.Address == "" {
			errs.Add("address", "укажите адрес")
		}
		if _, ok := FindDistrict(o.District); !ok {
			errs.Add("district", "выберите район")
		}
		return
	}

	o.Address = ""
	p := b.pickupPoint
	switch {
	case p == nil:
		if !errs.Has("pickupPointId") {
			errs.Add("pickupPointId", "выберите точку выдачи")
		}
	case p.Kind != method.Type():
		errs.Add("pickupPointId", fmt.Sprintf("«%s» не подходит для способа «%s»", p.Name, method.Title()))
	default:
		o.PickupPointID = p.ID
		o.District = p.District
	}
}

func validateExtras(codes []string, declared Money, errs *ValidationError) []string {
	result := make([]string, 0, len(codes))
	for _, code := range codes {
		if slices.Contains(result, code) {
			continue
		}
		if _, ok := FindExtra(code); !ok {
			errs.Add("extras", fmt.Sprintf("неизвестная услуга %q", code))
			continue
		}
		result = append(result, code)
	}
	if slices.Contains(result, ExtraInsurance) && declared <= 0 {
		errs.Add("declaredValue", "для страховки укажите объявленную ценность")
	}
	return result
}
