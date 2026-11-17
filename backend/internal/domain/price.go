package domain

import "slices"

// ПАТТЕРН DECORATOR — дополнительные услуги.
//
// Component         — PriceComponent.
// ConcreteComponent — basePrice (перевозка по стратегии + тариф доставки).
// Decorators        — insurance, fragile, express.
//
// Каждая услуга «оборачивает» предыдущую цену и добавляет своё.
// Услуги можно комбинировать в любом наборе.

type PriceLine struct {
	Label  string `json:"label"`
	Amount Money  `json:"amount"`
}

// Component
type PriceComponent interface {
	Total() Money
	Lines() []PriceLine
}

// ConcreteComponent
type basePrice struct {
	strategy PricingStrategy
	delivery DeliveryMethod
	weight   float64
	distance float64
}

func (b basePrice) transport() Money { return b.strategy.Calculate(b.weight, b.distance) }

func (b basePrice) Total() Money { return b.transport() + b.delivery.BaseFee() }

func (b basePrice) Lines() []PriceLine {
	return []PriceLine{
		{"Перевозка (" + b.strategy.Title() + ")", b.transport()},
		{"Тариф «" + b.delivery.Title() + "»", b.delivery.BaseFee()},
	}
}

// Decorator: страховка — 2% от объявленной ценности, минимум 200 ₸.
type insurance struct {
	inner    PriceComponent
	declared Money
}

func (e insurance) fee() Money   { return max(200, e.declared.Percent(2)) }
func (e insurance) Total() Money { return e.inner.Total() + e.fee() }
func (e insurance) Lines() []PriceLine {
	return append(e.inner.Lines(), PriceLine{"Страховка (2% от ценности)", e.fee()})
}

// Decorator: хрупкое — защитная упаковка, 400 ₸.
type fragile struct {
	inner PriceComponent
}

func (e fragile) Total() Money { return e.inner.Total() + 400 }
func (e fragile) Lines() []PriceLine {
	return append(e.inner.Lines(), PriceLine{"Хрупкое: защитная упаковка", 400})
}

// Decorator: срочно — +50% ко всей сумме под ним.
type express struct {
	inner PriceComponent
}

func (e express) fee() Money   { return e.inner.Total().Percent(50) }
func (e express) Total() Money { return e.inner.Total() + e.fee() }
func (e express) Lines() []PriceLine {
	return append(e.inner.Lines(), PriceLine{"Срочная доставка (+50%)", e.fee()})
}

const (
	ExtraInsurance = "insurance"
	ExtraFragile   = "fragile"
	ExtraExpress   = "express"
)

type Extra struct {
	Code, Title, Hint string
}

// Extras — доп. услуги в порядке применения (срочность последней,
// чтобы +50% считались от всей суммы).
var Extras = []Extra{
	{ExtraInsurance, "Страховка", "2% от объявленной ценности, мин. 200 ₸"},
	{ExtraFragile, "Хрупкое", "защитная упаковка, 400 ₸"},
	{ExtraExpress, "Срочная доставка", "+50% к итоговой сумме"},
}

// wrap оборачивает цену одной услугой.
func wrap(price PriceComponent, code string, declared Money) PriceComponent {
	switch code {
	case ExtraInsurance:
		return insurance{price, declared}
	case ExtraFragile:
		return fragile{price}
	case ExtraExpress:
		return express{price}
	}
	return price
}

// PriceInput — всё, что нужно для расчёта цены.
type PriceInput struct {
	Delivery      DeliveryType
	Strategy      string
	WeightKg      float64
	DistanceKm    float64
	DeclaredValue Money
	Extras        []string
}

// CalculatePrice: базовая цена по стратегии, затем обёртки выбранных услуг.
func CalculatePrice(in PriceInput) (PriceComponent, error) {
	delivery, err := NewDeliveryMethod(in.Delivery)
	if err != nil {
		return nil, err
	}
	strategy, err := PricingStrategyByCode(in.Strategy)
	if err != nil {
		return nil, err
	}

	var price PriceComponent = basePrice{strategy, delivery, in.WeightKg, in.DistanceKm}
	for _, extra := range Extras {
		if slices.Contains(in.Extras, extra.Code) {
			price = wrap(price, extra.Code, in.DeclaredValue)
		}
	}
	return price, nil
}
