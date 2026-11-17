package domain

import (
	"errors"
	"testing"
	"time"
)

var now = time.Date(2026, 11, 11, 10, 0, 0, 0, time.Local)

func locker() *PickupPoint {
	return &PickupPoint{ID: "PVZ-1", Name: "Постамат", District: "almaly", Kind: DeliveryLocker}
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
	if order.Status != StatusCreated || order.District != "almaly" {
		t.Fatalf("неверный заказ: %+v", order)
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
		build *OrderBuilder
		field string
	}{
		{"пустое описание", validLockerOrder().Parcel("  ", 1, 0), "description"},
		{"отрицательный вес", validLockerOrder().Parcel("Книги", -2, 0), "weightKg"},
		{"больше 15 кг в постамат", validLockerOrder().Parcel("Гантели", 15.1, 0), "weightKg"},
		{"страховка без ценности", validLockerOrder().Parcel("Книги", 1, 0).WithExtras(ExtraInsurance), "declaredValue"},
		{"до двери без адреса", validLockerOrder().Delivery(DeliveryDoor).ToAddress("", "almaly"), "address"},
		{"ПВЗ вместо постамата", validLockerOrder().ToPickupPoint(&PickupPoint{Kind: DeliveryPickupPoint}), "pickupPointId"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.build.Build(now)
			var ve ValidationError
			if !errors.As(err, &ve) || ve[tc.field] == "" {
				t.Fatalf("ожидали ошибку поля %q, получили %v", tc.field, err)
			}
		})
	}

	// Граница: ровно 15 кг в постамат — можно.
	if _, err := validLockerOrder().Parcel("Гантели", 15, 0).Build(now); err != nil {
		t.Fatalf("15 кг должны проходить: %v", err)
	}
}

// Strategy: одна посылка — разная цена при разных стратегиях.
func TestPricingStrategiesGiveDifferentPrices(t *testing.T) {
	in := PriceInput{Delivery: DeliveryDoor, WeightKg: 1, DistanceKm: 20}

	in.Strategy = "by_weight"
	byWeight, _ := CalculatePrice(in) // 400+150 + 1000
	in.Strategy = "by_distance"
	byDistance, _ := CalculatePrice(in) // 300+80*20 + 1000

	if byWeight.Total() != 1550 || byDistance.Total() != 2900 {
		t.Fatalf("по весу %d (ждали 1550), по расстоянию %d (ждали 2900)", byWeight.Total(), byDistance.Total())
	}
}

// Decorator: услуги оборачивают цену, срочность считается от всей суммы.
func TestExtrasDecoratePrice(t *testing.T) {
	price, err := CalculatePrice(PriceInput{
		Delivery: DeliveryLocker, Strategy: "by_weight", WeightKg: 1,
		DeclaredValue: 50000, Extras: []string{ExtraExpress, ExtraInsurance},
	})
	if err != nil {
		t.Fatal(err)
	}
	// база 550 + 300 = 850; страховка 1000 → 1850; срочно +50% → 2775
	if price.Total() != 2775 || len(price.Lines()) != 4 {
		t.Fatalf("итог %d (ждали 2775), строк %d (ждали 4)", price.Total(), len(price.Lines()))
	}
}

// Factory Method: курьер до двери не проходит через ПВЗ.
func TestStatusesDependOnDeliveryMethod(t *testing.T) {
	order, _ := validLockerOrder().Delivery(DeliveryDoor).ToAddress("Абая 1", "almaly").Build(now)
	courier := &Courier{ID: "CUR-1", Capacity: 1}

	if err := order.ChangeStatus(StatusInTransit, now); !errors.Is(err, ErrRule) {
		t.Fatalf("из «Создан» сразу в «В пути» нельзя, получили %v", err)
	}
	if err := order.AssignCourier(courier, now); err != nil {
		t.Fatal(err)
	}
	if err := order.ChangeStatus(StatusInTransit, now); err != nil {
		t.Fatal(err)
	}
	if err := order.ChangeStatus(StatusAtPickupPoint, now); !errors.Is(err, ErrRule) {
		t.Fatalf("доставка до двери не идёт через ПВЗ, получили %v", err)
	}
	if err := order.ChangeStatus(StatusDelivered, now); err != nil {
		t.Fatal(err)
	}
}

// Strategy: разные стратегии выбирают разных курьеров.
func TestCourierSelectorsChooseDifferently(t *testing.T) {
	order := &Order{District: "medeu"}
	couriers := []*Courier{
		{ID: "near", District: "medeu", Capacity: 5, ActiveOrders: 3},
		{ID: "free", District: "nauryzbay", Capacity: 5, ActiveOrders: 0},
		{ID: "full", District: "medeu", Capacity: 1, ActiveOrders: 1},
	}

	nearest, _ := NearestCourier{}.Select(order, couriers)
	leastLoaded, _ := LeastLoadedCourier{}.Select(order, couriers)

	if nearest.ID != "near" || leastLoaded.ID != "free" {
		t.Fatalf("ближайший = %s (ждали near), наименее загруженный = %s (ждали free)", nearest.ID, leastLoaded.ID)
	}
	if _, err := (NearestCourier{}).Select(order, couriers[2:]); !errors.Is(err, ErrRule) {
		t.Fatalf("без свободных курьеров ожидали ошибку, получили %v", err)
	}
}
