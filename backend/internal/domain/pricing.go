package domain

import "math"

// ПАТТЕРН STRATEGY — расчёт стоимости перевозки.
//
// Strategy         — PricingStrategy.
// ConcreteStrategy — WeightPricing, DistancePricing.
// Пользователь выбирает стратегию в форме заказа.

type PricingStrategy interface {
	Code() string
	Title() string
	Calculate(weightKg, distanceKm float64) Money
}

// WeightPricing: 400 ₸ + 150 ₸ за каждый начатый килограмм.
type WeightPricing struct{}

func (WeightPricing) Code() string  { return "by_weight" }
func (WeightPricing) Title() string { return "По весу: 400 ₸ + 150 ₸/кг" }
func (WeightPricing) Calculate(weightKg, _ float64) Money {
	return 400 + 150*Money(math.Ceil(weightKg))
}

// DistancePricing: 300 ₸ + 80 ₸ за каждый начатый километр.
type DistancePricing struct{}

func (DistancePricing) Code() string  { return "by_distance" }
func (DistancePricing) Title() string { return "По расстоянию: 300 ₸ + 80 ₸/км" }
func (DistancePricing) Calculate(_, distanceKm float64) Money {
	return 300 + 80*Money(math.Ceil(distanceKm))
}

// PricingStrategies — все стратегии. Новая стратегия добавляется сюда.
var PricingStrategies = []PricingStrategy{WeightPricing{}, DistancePricing{}}

func PricingStrategyByCode(code string) (PricingStrategy, error) {
	for _, s := range PricingStrategies {
		if s.Code() == code {
			return s, nil
		}
	}
	return nil, RuleViolation("неизвестный способ расчёта цены %q", code)
}
