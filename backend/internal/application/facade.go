// Package application — ПАТТЕРН FACADE.
//
// Facade    — DeliveryFacade: единственная точка входа для HTTP.
// Subsystem — команды (command), запросы (query), хранилище, наблюдатели.
//
// HTTP вызывает один метод, например CreateOrder. Фасад внутри выполняет
// команду (изменить данные), а затем запрос (прочитать результат для ответа).
package application

import (
	"encoding/json"
	"time"

	"parcelhub/internal/application/events"
	"parcelhub/internal/application/port"
	"parcelhub/internal/application/use_cases/command"
	"parcelhub/internal/application/use_cases/query"
	"parcelhub/internal/domain"
)

// Deps — всё, что нужно приложению снаружи.
type Deps struct {
	Orders       port.OrderRepository
	Customers    port.CustomerRepository
	Couriers     port.CourierRepository
	PickupPoints port.PickupPointRepository
	Activity     port.ActivityRepository
	Exports      port.ExportStore
	Publisher    *events.Publisher
}

type DeliveryFacade struct {
	orderCmd    *command.OrderCommands
	customerCmd *command.CustomerCommands
	courierCmd  *command.CourierCommands
	orders      *query.OrderQueries
	directory   *query.DirectoryQueries
	exports     port.ExportStore
}

func NewDeliveryFacade(d Deps) *DeliveryFacade {
	return &DeliveryFacade{
		orderCmd:    command.NewOrderCommands(d.Orders, d.Customers, d.Couriers, d.PickupPoints, d.Publisher, time.Now),
		customerCmd: command.NewCustomerCommands(d.Customers, d.Orders, time.Now),
		courierCmd:  command.NewCourierCommands(d.Couriers, time.Now),
		orders:      query.NewOrderQueries(d.Orders, d.Customers, d.Couriers, d.PickupPoints),
		directory:   query.NewDirectoryQueries(d.Customers, d.Couriers, d.PickupPoints, d.Orders, d.Activity),
		exports:     d.Exports,
	}
}

func (f *DeliveryFacade) Options() query.Options { return query.CatalogOptions() }

// --- Заказы ---

func (f *DeliveryFacade) ListOrders(status, search string) ([]query.OrderView, error) {
	return f.orders.List(status, search)
}

func (f *DeliveryFacade) GetOrder(id string) (query.OrderView, error) { return f.orders.Get(id) }

func (f *DeliveryFacade) QuoteOrder(in query.QuoteInput) ([]query.QuoteOption, error) {
	return f.orders.Quote(in)
}

func (f *DeliveryFacade) CreateOrder(in command.OrderInput) (query.OrderView, error) {
	id, err := f.orderCmd.Create(in)
	if err != nil {
		return query.OrderView{}, err
	}
	return f.orders.Get(id)
}

func (f *DeliveryFacade) UpdateOrder(id string, in command.OrderInput) (query.OrderView, error) {
	if err := f.orderCmd.Update(id, in); err != nil {
		return query.OrderView{}, err
	}
	return f.orders.Get(id)
}

func (f *DeliveryFacade) DeleteOrder(id string) error { return f.orderCmd.Delete(id) }

func (f *DeliveryFacade) AssignCourier(id, selector string) (query.OrderView, error) {
	if err := f.orderCmd.AssignCourier(id, selector); err != nil {
		return query.OrderView{}, err
	}
	return f.orders.Get(id)
}

func (f *DeliveryFacade) ChangeOrderStatus(id, status string) (query.OrderView, error) {
	if err := f.orderCmd.ChangeStatus(id, status); err != nil {
		return query.OrderView{}, err
	}
	return f.orders.Get(id)
}

// --- Клиенты ---

func (f *DeliveryFacade) ListCustomers() ([]query.CustomerView, error) {
	return f.directory.Customers()
}

func (f *DeliveryFacade) CreateCustomer(in command.CustomerInput) (query.CustomerView, error) {
	id, err := f.customerCmd.Create(in)
	if err != nil {
		return query.CustomerView{}, err
	}
	return f.directory.Customer(id)
}

func (f *DeliveryFacade) UpdateCustomer(id string, in command.CustomerInput) (query.CustomerView, error) {
	if err := f.customerCmd.Update(id, in); err != nil {
		return query.CustomerView{}, err
	}
	return f.directory.Customer(id)
}

func (f *DeliveryFacade) DeleteCustomer(id string) error { return f.customerCmd.Delete(id) }

// --- Курьеры ---

func (f *DeliveryFacade) ListCouriers() ([]query.CourierView, error) { return f.directory.Couriers() }

func (f *DeliveryFacade) CreateCourier(in command.CourierInput) (query.CourierView, error) {
	id, err := f.courierCmd.Create(in)
	if err != nil {
		return query.CourierView{}, err
	}
	return f.directory.Courier(id)
}

func (f *DeliveryFacade) UpdateCourier(id string, in command.CourierInput) (query.CourierView, error) {
	if err := f.courierCmd.Update(id, in); err != nil {
		return query.CourierView{}, err
	}
	return f.directory.Courier(id)
}

func (f *DeliveryFacade) DeleteCourier(id string) error { return f.courierCmd.Delete(id) }

// --- Точки выдачи, журнал, отчёт ---

func (f *DeliveryFacade) ListPickupPoints() ([]query.PickupPointView, error) {
	return f.directory.PickupPoints()
}

func (f *DeliveryFacade) Activity(limit int) ([]port.ActivityEntry, error) {
	return f.directory.Activity(limit)
}

func (f *DeliveryFacade) DailyReport(day time.Time) (query.DailyReport, error) {
	return f.orders.DailyReport(day)
}

// ExportDailyReport — отчёт в CSV или JSON; копия сохраняется на диск.
// Возвращает имя файла и содержимое.
func (f *DeliveryFacade) ExportDailyReport(day time.Time, format string) (string, []byte, error) {
	report, err := f.orders.DailyReport(day)
	if err != nil {
		return "", nil, err
	}

	var data []byte
	switch format {
	case "csv":
		data = report.CSV()
	case "json":
		data, _ = json.MarshalIndent(report, "", "  ")
	default:
		return "", nil, domain.RuleViolation("формат выгрузки: csv или json")
	}

	name := "report-" + report.Date + "." + format
	if err := f.exports.SaveExport(name, data); err != nil {
		return "", nil, err
	}
	return name, data, nil
}
