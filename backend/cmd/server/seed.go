package main

import (
	"strings"

	"parcelhub/internal/adapters/out/storage"
	"parcelhub/internal/application"
	"parcelhub/internal/application/use_cases/command"
	"parcelhub/internal/domain"
)

// seed заполняет пустое хранилище тестовыми данными, чтобы приложение
// можно было показать сразу. Заказы создаются через фасад, как из интерфейса.
func seed(f *application.DeliveryFacade, store *storage.Store) error {
	points := []struct {
		name, address, district string
		kind                    domain.DeliveryType
	}{
		{"ПВЗ «Абая»", "пр. Абая, 52", "almaly", domain.DeliveryPickupPoint},
		{"ПВЗ «Сайран»", "ул. Толе би, 286", "auezov", domain.DeliveryPickupPoint},
		{"Постамат «Mega»", "ул. Розыбакиева, 247", "bostandyk", domain.DeliveryLocker},
		{"Постамат «Dostyk Plaza»", "пр. Достык, 111", "medeu", domain.DeliveryLocker},
	}
	pointIDs := map[string]string{}
	for _, p := range points {
		point, err := domain.NewPickupPoint(p.name, p.address, p.district, p.kind)
		if err != nil {
			return err
		}
		if err := store.PickupPoints.Save(point); err != nil {
			return err
		}
		pointIDs[p.name] = point.ID
	}

	customers := []command.CustomerInput{
		{Name: "Айгерим Садыкова", Phone: "+77011234567"},
		{Name: "Данияр Ахметов", Phone: "+77027654321"},
		{Name: "Мария Ким", Phone: "+77055551122"},
		{Name: "Ерлан Жумабаев", Phone: "+77079990011"},
		{Name: "Алина Петрова", Phone: "+77471234590"},
	}
	customerIDs := make([]string, len(customers))
	for i, in := range customers {
		c, err := f.CreateCustomer(in)
		if err != nil {
			return err
		}
		customerIDs[i] = c.ID
	}

	couriers := []command.CourierInput{
		{Name: "Асхат Нурланов", Phone: "+77015550001", District: "almaly", Capacity: 3},
		{Name: "Олжас Серикбаев", Phone: "+77015550002", District: "auezov", Capacity: 4},
		{Name: "Игорь Ли", Phone: "+77015550003", District: "medeu", Capacity: 2},
		{Name: "Бекзат Омаров", Phone: "+77015550004", District: "turksib", Capacity: 3},
	}
	for _, in := range couriers {
		if _, err := f.CreateCourier(in); err != nil {
			return err
		}
	}

	type seedOrder struct {
		in    command.OrderInput
		steps []string // "assign:<selector>" или статус
	}
	orders := []seedOrder{
		{in: command.OrderInput{
			CustomerID: customerIDs[0], Description: "Ноутбук Lenovo", WeightKg: 2.4, DeclaredValue: 450000,
			DeliveryType: "door", Address: "мкр. Самал-2, д. 58, кв. 14", District: "medeu",
			PricingStrategy: "by_distance", Extras: []string{"insurance", "fragile"},
		}, steps: []string{"assign:nearest", "in_transit", "delivered"}},
		{in: command.OrderInput{
			CustomerID: customerIDs[1], Description: "Книги, 6 шт.", WeightKg: 4,
			DeliveryType: "pickup_point", PickupPointID: pointIDs["ПВЗ «Сайран»"], PricingStrategy: "by_weight",
		}, steps: []string{"assign:least_loaded", "in_transit", "at_pickup_point"}},
		{in: command.OrderInput{
			CustomerID: customerIDs[2], Description: "Косметика", WeightKg: 0.8, DeclaredValue: 25000,
			DeliveryType: "locker", PickupPointID: pointIDs["Постамат «Mega»"], PricingStrategy: "by_weight",
			Extras: []string{"express"},
		}, steps: []string{"assign:nearest", "in_transit"}},
		{in: command.OrderInput{
			CustomerID: customerIDs[3], Description: "Велосипедные колёса", WeightKg: 12,
			DeliveryType: "door", Address: "ул. Сейфуллина, 410", District: "turksib", PricingStrategy: "by_distance",
		}, steps: []string{"assign:nearest"}},
		{in: command.OrderInput{
			CustomerID: customerIDs[4], Description: "Детская одежда", WeightKg: 1.5,
			DeliveryType: "pickup_point", PickupPointID: pointIDs["ПВЗ «Абая»"], PricingStrategy: "by_weight",
		}},
		{in: command.OrderInput{
			CustomerID: customerIDs[0], Description: "Ваза керамическая", WeightKg: 3, DeclaredValue: 30000,
			DeliveryType: "door", Address: "ул. Жандосова, 98", District: "bostandyk", PricingStrategy: "by_weight",
			Extras: []string{"fragile", "insurance"},
		}},
		{in: command.OrderInput{
			CustomerID: customerIDs[1], Description: "Наушники", WeightKg: 0.4, DeclaredValue: 60000,
			DeliveryType: "locker", PickupPointID: pointIDs["Постамат «Dostyk Plaza»"], PricingStrategy: "by_distance",
		}, steps: []string{"cancelled"}},
	}

	for _, o := range orders {
		created, err := f.CreateOrder(o.in)
		if err != nil {
			return err
		}
		for _, step := range o.steps {
			if selector, ok := strings.CutPrefix(step, "assign:"); ok {
				_, err = f.AssignCourier(created.ID, selector)
			} else {
				_, err = f.ChangeOrderStatus(created.ID, step)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}
