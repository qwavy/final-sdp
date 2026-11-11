package domain

import "slices"

// ПАТТЕРН DECORATOR (дополнительные услуги)
//
// Component          — PriceComponent (итог + строки расшифровки цены).
// ConcreteComponent  — basePrice: перевозка по выбранной стратегии + тариф доставки.
// Decorators         — insuranceExtra, fragileExtra, expressExtra.
//
// Каждая услуга «оборачивает» предыдущую цену и добавляет к ней своё.
// Услуги комбинируются в любом наборе, а базовый расчёт об этом не знает.
// Новая услуга = новый тип-обёртка + одна запись в extras.

type PriceLine struct {
	Label  string `json:"label"`
	Amount Money  `json:"amount"`
}

// PriceComponent — Component.
type PriceComponent interface {
	Total() Money
	Lines() []PriceLine
}

// basePrice — ConcreteComponent.
type basePrice struct {
	strategy PricingStrategy
	method   DeliveryMethod
	input    PricingInput
}

func (b basePrice) Total() Money {
	return b.strategy.Calculate(b.input) + b.method.BaseFee()
}

func (b basePrice) Lines() []PriceLine {
	return []PriceLine{
		{Label: "Перевозка (" + b.strategy.Title() + ")", Amount: b.strategy.Calculate(b.input)},
		{Label: "Тариф «" + b.method.Title() + "»", Amount: b.method.BaseFee()},
	}
}

// insuranceExtra — страховка: 2% от объявленной ценности, минимум 200 ₸.
type insuranceExtra struct {
	inner    PriceComponent
	declared Money
}

func (e insuranceExtra) fee() Money   { return max(200, e.declared.Percent(2)) }
func (e insuranceExtra) Total() Money { return e.inner.Total() + e.fee() }
func (e insuranceExtra) Lines() []PriceLine {
	return withLine(e.inner, "Страховка (2% от ценности)", e.fee())
}

// fragileExtra — хрупкое: защитная упаковка, фиксированно 400 ₸.
type fragileExtra struct {
	inner PriceComponent
}

func (e fragileExtra) Total() Money { return e.inner.Total() + 400 }
func (e fragileExtra) Lines() []PriceLine {
	return withLine(e.inner, "Хрупкое: защитная упаковка", 400)
}

// expressExtra — срочная доставка: +50% ко всей сумме, которую оборачивает.
type expressExtra struct {
	inner PriceComponent
}

func (e expressExtra) fee() Money   { return e.inner.Total().Percent(50) }
func (e expressExtra) Total() Money { return e.inner.Total() + e.fee() }
func (e expressExtra) Lines() []PriceLine {
	return withLine(e.inner, "Срочная доставка (+50%)", e.fee())
}

func withLine(inner PriceComponent, label string, amount Money) []PriceLine {
	return append(slices.Clone(inner.Lines()), PriceLine{Label: label, Amount: amount})
}

// Extra — описание доп. услуги и функция, которая оборачивает цену.
type Extra struct {
	Code  string
	Title string
	Hint  string
	wrap  func(inner PriceComponent, declared Money) PriceComponent
}

const (
	ExtraInsurance = "insurance"
	ExtraFragile   = "fragile"
	ExtraExpress   = "express"
)

// Порядок применения: срочность последней, чтобы +50% считались от всего остального.
var extras = []Extra{
	{
		Code: ExtraInsurance, Title: "Страховка", Hint: "2% от объявленной ценности, мин. 200 ₸",
		wrap: func(inner PriceComponent, declared Money) PriceComponent { return insuranceExtra{inner, declared} },
	},
	{
		Code: ExtraFragile, Title: "Хрупкое", Hint: "защитная упаковка, 400 ₸",
		wrap: func(inner PriceComponent, _ Money) PriceComponent { return fragileExtra{inner} },
	},
	{
		Code: ExtraExpress, Title: "Срочная доставка", Hint: "+50% к итоговой сумме",
		wrap: func(inner PriceComponent, _ Money) PriceComponent { return expressExtra{inner} },
	},
}

func Extras() []Extra { return append([]Extra(nil), extras...) }

func FindExtra(code string) (Extra, bool) {
	for _, e := range extras {
		if e.Code == code {
			return e, true
		}
	}
	return Extra{}, false
}

// QuoteSpec — всё, что нужно для расчёта цены.
type QuoteSpec struct {
	DeliveryType  DeliveryType
	PricingCode   string
	WeightKg      float64
	DistanceKm    float64
	DeclaredValue Money
	Extras        []string
}

// BuildPrice собирает цену: базовый расчёт по стратегии, затем обёртки доп. услуг.
func BuildPrice(spec QuoteSpec) (PriceComponent, error) {
	method, err := NewDeliveryMethod(spec.DeliveryType)
	if err != nil {
		return nil, err
	}
	strategy, err := PricingStrategyByCode(spec.PricingCode)
	if err != nil {
		return nil, err
	}
	for _, code := range spec.Extras {
		if _, ok := FindExtra(code); !ok {
			return nil, RuleViolation("неизвестная услуга %q", code)
		}
	}

	var price PriceComponent = basePrice{
		strategy: strategy,
		method:   method,
		input:    PricingInput{WeightKg: spec.WeightKg, DistanceKm: spec.DistanceKm},
	}
	for _, extra := range extras {
		if slices.Contains(spec.Extras, extra.Code) {
			price = extra.wrap(price, spec.DeclaredValue)
		}
	}
	return price, nil
}
