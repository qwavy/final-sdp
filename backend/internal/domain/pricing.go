package domain

import "math"

// ПАТТЕРН STRATEGY (расчёт стоимости перевозки)
//
// Strategy          — PricingStrategy.
// ConcreteStrategy  — WeightPricing, DistancePricing.
// Context           — basePrice (см. price.go): берёт стратегию, которую выбрал пользователь.
//
// Пользователь выбирает стратегию в форме заказа. Новая стратегия =
// новый тип с методами Code/Title/Calculate + одна строка в pricingStrategies.

type PricingInput struct {
	WeightKg   float64
	DistanceKm float64
}

type PricingStrategy interface {
	Code() string
	Title() string
	Calculate(in PricingInput) Money
}

// WeightPricing — цена зависит от веса: 400 ₸ + 150 ₸ за каждый начатый кг.
type WeightPricing struct{}

func (WeightPricing) Code() string  { return "by_weight" }
func (WeightPricing) Title() string { return "По весу: 400 ₸ + 150 ₸/кг" }
func (WeightPricing) Calculate(in PricingInput) Money {
	return 400 + 150*Money(math.Ceil(in.WeightKg))
}

// DistancePricing — цена зависит от расстояния: 300 ₸ + 80 ₸ за каждый начатый км.
type DistancePricing struct{}

func (DistancePricing) Code() string  { return "by_distance" }
func (DistancePricing) Title() string { return "По расстоянию: 300 ₸ + 80 ₸/км" }
func (DistancePricing) Calculate(in PricingInput) Money {
	return 300 + 80*Money(math.Ceil(in.DistanceKm))
}

var pricingStrategies = []PricingStrategy{
	WeightPricing{},
	DistancePricing{},
}

func PricingStrategies() []PricingStrategy {
	return append([]PricingStrategy(nil), pricingStrategies...)
}

func PricingStrategyByCode(code string) (PricingStrategy, error) {
	for _, s := range pricingStrategies {
		if s.Code() == code {
			return s, nil
		}
	}
	return nil, RuleViolation("неизвестный способ расчёта цены %q", code)
}
