// Package application — ПАТТЕРН FACADE.
//
// Facade     — DeliveryFacade: единственная точка входа для интерфейса (HTTP).
// Subsystem  — команды (command), запросы (query), репозитории, наблюдатели.
//
// HTTP-обработчик вызывает один метод, например CreateOrder, а фасад внутри
// выполняет команду, затем запрос для показа результата. Интерфейс не знает
// о командах, запросах и хранилище — их можно менять, не трогая HTTP.
package application

import (
	"fmt"
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
	Events       *events.Publisher
	Clock        func() time.Time
}

type DeliveryFacade struct {
	orderCmd    *command.OrderCommands
	customerCmd *command.CustomerCommands
	courierCmd  *command.CourierCommands
	orders      *query.OrderQueries
	directory   *query.DirectoryQueries
	reports     *query.ReportQueries
	exports     port.ExportStore
}

func NewDeliveryFacade(d Deps) *DeliveryFacade {
	if d.Clock == nil {
		d.Clock = time.Now
	}
	return &DeliveryFacade{
		orderCmd:    command.NewOrderCommands(d.Orders, d.Customers, d.Couriers, d.PickupPoints, d.Events, d.Clock),
		customerCmd: command.NewCustomerCommands(d.Customers, d.Orders, d.Clock),
		courierCmd:  command.NewCourierCommands(d.Couriers, d.Clock),
		orders:      query.NewOrderQueries(d.Orders, d.Customers, d.Couriers, d.PickupPoints),
		directory:   query.NewDirectoryQueries(d.Customers, d.Couriers, d.PickupPoints, d.Orders, d.Activity),
		reports:     query.NewReportQueries(d.Orders, d.Customers, d.Couriers, d.PickupPoints, d.Clock),
		exports:     d.Exports,
	}
}

// --- Справочники ---

func (f *DeliveryFacade) Options() query.Options { return query.CatalogOptions() }

// --- Заказы ---

func (f *DeliveryFacade) ListOrders(filter query.OrderFilter) ([]query.OrderView, error) {
	return f.orders.List(filter)
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

func (f *DeliveryFacade) AssignCourier(orderID, selector string) (query.OrderView, error) {
	if err := f.orderCmd.AssignCourier(orderID, selector); err != nil {
		return query.OrderView{}, err
	}
	return f.orders.Get(orderID)
}

func (f *DeliveryFacade) ChangeOrderStatus(orderID, status string) (query.OrderView, error) {
	if err := f.orderCmd.ChangeStatus(orderID, status); err != nil {
		return query.OrderView{}, err
	}
	return f.orders.Get(orderID)
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

// --- Точки выдачи и журнал ---

func (f *DeliveryFacade) ListPickupPoints() ([]query.PickupPointView, error) {
	return f.directory.PickupPoints()
}

func (f *DeliveryFacade) Activity(limit int) ([]port.ActivityEntry, error) {
	return f.directory.Activity(limit)
}

// --- Отчёт ---

func (f *DeliveryFacade) DailyReport(day time.Time) (query.DailyReport, error) {
	return f.reports.Daily(day)
}

type ExportedFile struct {
	Name        string
	ContentType string
	Data        []byte
	SavedTo     string
}

// ExportDailyReport строит отчёт, переводит в CSV/JSON и сохраняет копию на диск.
func (f *DeliveryFacade) ExportDailyReport(day time.Time, format string) (ExportedFile, error) {
	report, err := f.reports.Daily(day)
	if err != nil {
		return ExportedFile{}, err
	}

	file := ExportedFile{Name: fmt.Sprintf("report-%s.%s", report.Date, format)}
	switch format {
	case "csv":
		file.ContentType = "text/csv; charset=utf-8"
		file.Data, err = report.CSV()
	case "json":
		file.ContentType = "application/json"
		file.Data, err = report.JSON()
	default:
		return ExportedFile{}, domain.RuleViolation("формат выгрузки: csv или json")
	}
	if err != nil {
		return ExportedFile{}, err
	}

	file.SavedTo, err = f.exports.SaveExport(file.Name, file.Data)
	return file, err
}
