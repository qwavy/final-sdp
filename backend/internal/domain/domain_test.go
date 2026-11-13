package domain

import (
	"errors"
	"testing"
	"time"
)

var now = time.Date(2026, 11, 11, 10, 0, 0, 0, time.Local)

func locker() *PickupPoint {
	return &PickupPoint{ID: "PVZ-1", Name: "Постамат", Address: "Абая 1", District: "almaly", Kind: DeliveryLocker}
}

func validLockerOrder() *OrderBuilder {
	return NewOrderBuilder().
		ForCustomer("CUS-1").
		Parcel("Книги", 2.5, 10000).
		Delivery(DeliveryLocker).
		ToPickupPoint(locker()).
		PricedBy("by_weight")
}

// Builder: корректный заказ собирается, цена и район вычисляются.
func TestBuilderBuildsValidOrder(t *testing.T) {
	order, err := validLockerOrder().WithExtras(ExtraFragile).Build(now)
	if err != nil {
		t.Fatalf("ожидали успех, получили %v", err)
	}
	if order.Status != StatusCreated || order.District != "almaly" || order.PickupPointID != "PVZ-1" {
		t.Fatalf("неверное состояние заказа: %+v", order)
	}
	// по весу: 400 + 150*3 = 850; постамат 300; хрупкое 400
	if order.Price != 1550 {
		t.Fatalf("цена = %d, ожидали 1550", order.Price)
	}
}

// Builder: неверный ввод и граничные случаи.
func TestBuilderRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name  string
		build func() *OrderBuilder
		field string
	}{
		{"пустое описание", func() *OrderBuilder {
			return validLockerOrder().Parcel("  ", 1, 0)
		}, "description"},
		{"отрицательный вес", func() *OrderBuilder {
			return validLockerOrder().Parcel("Книги", -2, 0)
		}, "weightKg"},
		{"вес больше лимита постамата (15 кг)", func() *OrderBuilder {
			return validLockerOrder().Parcel("Гантели", 15.1, 0)
		}, "weightKg"},
		{"страховка без объявленной ценности", func() *OrderBuilder {
			return validLockerOrder().Parcel("Книги", 1, 0).WithExtras(ExtraInsurance)
		}, "declaredValue"},
		{"доставка до двери без адреса", func() *OrderBuilder {
			return validLockerOrder().Delivery(DeliveryDoor).ToAddress("", "almaly")
		}, "address"},
		{"пункт выдачи вместо постамата", func() *OrderBuilder {
			p := locker()
			p.Kind = DeliveryPickupPoint
			return validLockerOrder().ToPickupPoint(p)
		}, "pickupPointId"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.build().Build(now)
			var ve *ValidationError
			if !errors.As(err, &ve) || !ve.Has(tc.field) {
				t.Fatalf("ожидали ошибку поля %q, получили %v", tc.field, err)
			}
		})
	}

	// Граница: ровно 15 кг для постамата — допустимо.
	if _, err := validLockerOrder().Parcel("Гантели", 15, 0).Build(now); err != nil {
		t.Fatalf("15 кг должны проходить: %v", err)
	}
}

// Strategy: одна посылка — разная цена при разных стратегиях.
func TestPricingStrategiesGiveDifferentPrices(t *testing.T) {
	spec := QuoteSpec{DeliveryType: DeliveryDoor, WeightKg: 1, DistanceKm: 20}

	spec.PricingCode = "by_weight"
	byWeight, _ := BuildPrice(spec) // 400+150 + 1000
	spec.PricingCode = "by_distance"
	byDistance, _ := BuildPrice(spec) // 300+80*20 + 1000

	if byWeight.Total() != 1550 || byDistance.Total() != 2900 {
		t.Fatalf("по весу %d (ждали 1550), по расстоянию %d (ждали 2900)", byWeight.Total(), byDistance.Total())
	}
}

// Decorator: услуги оборачивают цену, срочность считается от всей суммы.
func TestExtrasDecoratePrice(t *testing.T) {
	price, err := BuildPrice(QuoteSpec{
		DeliveryType: DeliveryLocker, PricingCode: "by_weight", WeightKg: 1,
		DeclaredValue: 50000, Extras: []string{ExtraExpress, ExtraInsurance},
	})
	if err != nil {
		t.Fatal(err)
	}
	// база 550 + 300 = 850; страховка 2% от 50000 = 1000 → 1850; срочно +50% → 2775
	if price.Total() != 2775 {
		t.Fatalf("итог %d, ожидали 2775", price.Total())
	}
	if n := len(price.Lines()); n != 4 {
		t.Fatalf("строк расшифровки %d, ожидали 4", n)
	}
}

// Factory Method + правила статусов: курьер до двери не проходит через ПВЗ.
func TestStatusTransitionsDependOnDeliveryMethod(t *testing.T) {
	order, _ := validLockerOrder().Delivery(DeliveryDoor).ToAddress("Абая 1", "almaly").Build(now)
	courier := &Courier{ID: "CUR-1", Name: "Асхат", Capacity: 1}

	if err := order.ChangeStatus(StatusInTransit, now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("из «Создан» сразу в «В пути» нельзя, получили %v", err)
	}
	if err := order.AssignCourier(courier, now); err != nil {
		t.Fatal(err)
	}
	if err := order.ChangeStatus(StatusInTransit, now); err != nil {
		t.Fatal(err)
	}
	if err := order.ChangeStatus(StatusAtPickupPoint, now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("доставка до двери не идёт через ПВЗ, получили %v", err)
	}
	if err := order.ChangeStatus(StatusDelivered, now); err != nil {
		t.Fatal(err)
	}
	if courier.ActiveOrders != 1 {
		t.Fatalf("у курьера должен быть 1 заказ, а не %d", courier.ActiveOrders)
	}
}

// Strategy: разные стратегии выбирают разных курьеров.
func TestCourierSelectorsChooseDifferently(t *testing.T) {
	order := &Order{District: "medeu"}
	couriers := []*Courier{
		{ID: "near", Name: "Рядом", District: "medeu", Capacity: 5, ActiveOrders: 3},
		{ID: "free", Name: "Свободный", District: "nauryzbay", Capacity: 5, ActiveOrders: 0},
		{ID: "full", Name: "Занятой", District: "medeu", Capacity: 1, ActiveOrders: 1},
	}

	nearest, _ := NearestCourierSelector{}.Select(order, couriers)
	leastLoaded, _ := LeastLoadedCourierSelector{}.Select(order, couriers)

	if nearest.ID != "near" || leastLoaded.ID != "free" {
		t.Fatalf("ближайший = %s (ждали near), наименее загруженный = %s (ждали free)", nearest.ID, leastLoaded.ID)
	}
	if _, err := (NearestCourierSelector{}).Select(order, couriers[2:]); !errors.Is(err, ErrRuleViolation) {
		t.Fatalf("без свободных курьеров ожидали ошибку, получили %v", err)
	}
}
