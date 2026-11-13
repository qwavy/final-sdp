package application_test

import (
	"errors"
	"testing"
	"time"

	"parcelhub/internal/adapters/out/storage"
	"parcelhub/internal/application"
	"parcelhub/internal/application/events"
	"parcelhub/internal/application/port"
	"parcelhub/internal/application/use_cases/command"
	"parcelhub/internal/domain"
)

// fakeNotifier запоминает уведомления вместо отправки SMS.
type fakeNotifier struct{ sent []port.Notification }

func (f *fakeNotifier) Notify(n port.Notification) error {
	f.sent = append(f.sent, n)
	return nil
}

type testApp struct {
	facade   *application.DeliveryFacade
	store    *storage.Store
	notifier *fakeNotifier
	customer string
	point    string
}

func newTestApp(t *testing.T, dir string) *testApp {
	t.Helper()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	notifier := &fakeNotifier{}
	publisher := events.NewPublisher()
	publisher.Subscribe(events.NewActivityRecorder(store.Activity))
	publisher.Subscribe(events.NewCustomerNotifier(store.Orders, store.Customers, store.PickupPoints, notifier, store.Activity))

	facade := application.NewDeliveryFacade(application.Deps{
		Orders: store.Orders, Customers: store.Customers, Couriers: store.Couriers,
		PickupPoints: store.PickupPoints, Activity: store.Activity, Exports: store.Exports,
		Events: publisher, Clock: time.Now,
	})
	return &testApp{facade: facade, store: store, notifier: notifier}
}

func (a *testApp) seedBasics(t *testing.T) {
	t.Helper()
	c, err := a.facade.CreateCustomer(command.CustomerInput{Name: "Айгерим", Phone: "8 701 123 45 67"})
	if err != nil {
		t.Fatal(err)
	}
	a.customer = c.ID
	if _, err := a.facade.CreateCourier(command.CourierInput{Name: "Асхат", Phone: "+77015550001", District: "almaly", Capacity: 2}); err != nil {
		t.Fatal(err)
	}
	point, _ := domain.NewPickupPoint("ПВЗ Абая", "Абая 52", "almaly", domain.DeliveryPickupPoint)
	if err := a.store.PickupPoints.Save(point); err != nil {
		t.Fatal(err)
	}
	a.point = point.ID
}

func (a *testApp) pickupOrder() command.OrderInput {
	return command.OrderInput{
		CustomerID: a.customer, Description: "Книги", WeightKg: 2,
		DeliveryType: "pickup_point", PickupPointID: a.point, PricingStrategy: "by_weight",
	}
}

// Основной сценарий: создать → назначить курьера → в пути → в ПВЗ → выдан.
// Заодно проверяем реакцию на события (Observer): SMS и журнал.
func TestMainJourneyAndEventReactions(t *testing.T) {
	app := newTestApp(t, t.TempDir())
	app.seedBasics(t)

	order, err := app.facade.CreateOrder(app.pickupOrder())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.facade.AssignCourier(order.ID, "nearest"); err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"in_transit", "at_pickup_point", "delivered"} {
		if _, err := app.facade.ChangeOrderStatus(order.ID, status); err != nil {
			t.Fatalf("статус %s: %v", status, err)
		}
	}

	final, _ := app.facade.GetOrder(order.ID)
	if final.Status != domain.StatusDelivered || final.CourierName != "Асхат" {
		t.Fatalf("неверный итог: %s, курьер %q", final.Status, final.CourierName)
	}
	couriers, _ := app.facade.ListCouriers()
	if couriers[0].ActiveOrders != 0 {
		t.Fatalf("после выдачи курьер должен освободиться, активных: %d", couriers[0].ActiveOrders)
	}

	// 4 смены статуса → 4 SMS на нормализованный номер
	if len(app.notifier.sent) != 4 || app.notifier.sent[0].Phone != "+77011234567" {
		t.Fatalf("SMS: %+v", app.notifier.sent)
	}
	activity, _ := app.facade.Activity(0)
	if len(activity) < 8 {
		t.Fatalf("в журнале мало записей: %d", len(activity))
	}

	report, _ := app.facade.DailyReport(time.Now())
	if report.TotalOrders != 1 || report.DeliveredOrders != 1 || report.Revenue != final.Price {
		t.Fatalf("неверный отчёт: %+v", report)
	}
}

// Сохранение и повторная загрузка: данные остаются после «перезапуска».
func TestDataSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	app := newTestApp(t, dir)
	app.seedBasics(t)
	order, err := app.facade.CreateOrder(app.pickupOrder())
	if err != nil {
		t.Fatal(err)
	}

	restarted := newTestApp(t, dir)
	reloaded, err := restarted.facade.GetOrder(order.ID)
	if err != nil {
		t.Fatalf("после перезапуска заказ не найден: %v", err)
	}
	if reloaded.Price != order.Price || reloaded.CustomerName != "Айгерим" {
		t.Fatalf("данные изменились: %+v", reloaded)
	}
}

// Нарушение правил: нельзя удалить клиента с активным заказом и
// нельзя изменить заказ после назначения курьера.
func TestBusinessRulesAreEnforced(t *testing.T) {
	app := newTestApp(t, t.TempDir())
	app.seedBasics(t)
	order, _ := app.facade.CreateOrder(app.pickupOrder())
	_, _ = app.facade.AssignCourier(order.ID, "least_loaded")

	if err := app.facade.DeleteCustomer(app.customer); !errors.Is(err, domain.ErrRuleViolation) {
		t.Fatalf("удаление клиента с активным заказом: ожидали отказ, получили %v", err)
	}
	if _, err := app.facade.UpdateOrder(order.ID, app.pickupOrder()); !errors.Is(err, domain.ErrRuleViolation) {
		t.Fatalf("изменение назначенного заказа: ожидали отказ, получили %v", err)
	}
	if _, err := app.facade.CreateCustomer(command.CustomerInput{Name: "Дубль", Phone: "+77011234567"}); err == nil {
		t.Fatal("клиент с повторяющимся телефоном не должен создаваться")
	}
}
