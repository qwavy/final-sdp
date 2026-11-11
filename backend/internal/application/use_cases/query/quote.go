package query

import (
	"parcelhub/internal/domain"
)

// QuoteInput — данные формы для предварительного расчёта цены.
type QuoteInput struct {
	DeliveryType  string   `json:"deliveryType"`
	WeightKg      float64  `json:"weightKg"`
	DeclaredValue int64    `json:"declaredValue"`
	District      string   `json:"district"`
	PickupPointID string   `json:"pickupPointId"`
	Extras        []string `json:"extras"`
}

// QuoteOption — цена по одной стратегии. Форма показывает все варианты рядом.
type QuoteOption struct {
	Strategy   string             `json:"strategy"`
	Title      string             `json:"title"`
	DistanceKm float64            `json:"distanceKm"`
	Total      domain.Money       `json:"total"`
	Lines      []domain.PriceLine `json:"lines"`
}

// Quote считает цену сразу по всем стратегиям, ничего не сохраняя.
func (q *OrderQueries) Quote(in QuoteInput) ([]QuoteOption, error) {
	var errs domain.ValidationError
	district := in.District

	method, err := domain.NewDeliveryMethod(domain.DeliveryType(in.DeliveryType))
	if err != nil {
		errs.Add("deliveryType", "выберите способ доставки")
	}
	if method != nil && method.NeedsPickupPoint() {
		point, err := q.pickupPoints.Get(in.PickupPointID)
		if err != nil {
			errs.Add("pickupPointId", "выберите точку выдачи")
		} else {
			district = point.District
		}
	}
	if in.WeightKg <= 0 {
		errs.Add("weightKg", "вес должен быть больше 0")
	}
	distance, err := domain.DistanceFromHub(district)
	if err != nil && !errs.Has("pickupPointId") {
		errs.Add("district", "выберите район")
	}
	if err := errs.OrNil(); err != nil {
		return nil, err
	}

	options := make([]QuoteOption, 0)
	for _, strategy := range domain.PricingStrategies() {
		price, err := domain.BuildPrice(domain.QuoteSpec{
			DeliveryType:  method.Type(),
			PricingCode:   strategy.Code(),
			WeightKg:      in.WeightKg,
			DistanceKm:    distance,
			DeclaredValue: domain.Money(in.DeclaredValue),
			Extras:        in.Extras,
		})
		if err != nil {
			return nil, err
		}
		options = append(options, QuoteOption{
			Strategy:   strategy.Code(),
			Title:      strategy.Title(),
			DistanceKm: distance,
			Total:      price.Total(),
			Lines:      price.Lines(),
		})
	}
	return options, nil
}
