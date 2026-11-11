package query

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"

	"parcelhub/internal/application/port"
	"parcelhub/internal/domain"
)

// DailyReport — итог работы за день: итоговый результат приложения.
type DailyReport struct {
	Date            string           `json:"date"`
	GeneratedAt     time.Time        `json:"generatedAt"`
	TotalOrders     int              `json:"totalOrders"`
	ActiveOrders    int              `json:"activeOrders"`
	DeliveredOrders int              `json:"deliveredOrders"`
	CancelledOrders int              `json:"cancelledOrders"`
	Revenue         domain.Money     `json:"revenue"`
	AveragePrice    domain.Money     `json:"averagePrice"`
	ByStatus        []StatusCount    `json:"byStatus"`
	ByDelivery      []DeliveryStat   `json:"byDelivery"`
	Couriers        []CourierLoad    `json:"couriers"`
	Orders          []ReportOrderRow `json:"orders"`
}

type StatusCount struct {
	Status string `json:"status"`
	Title  string `json:"title"`
	Count  int    `json:"count"`
}

type DeliveryStat struct {
	Type    string       `json:"type"`
	Title   string       `json:"title"`
	Count   int          `json:"count"`
	Revenue domain.Money `json:"revenue"`
}

type CourierLoad struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ActiveOrders int    `json:"activeOrders"`
	Capacity     int    `json:"capacity"`
	Delivered    int    `json:"delivered"`
}

type ReportOrderRow struct {
	ID       string       `json:"id"`
	Created  time.Time    `json:"created"`
	Customer string       `json:"customer"`
	Delivery string       `json:"delivery"`
	Status   string       `json:"status"`
	Courier  string       `json:"courier"`
	Price    domain.Money `json:"price"`
}

type ReportQueries struct {
	orders       port.OrderRepository
	customers    port.CustomerRepository
	couriers     port.CourierRepository
	pickupPoints port.PickupPointRepository
	now          func() time.Time
}

func NewReportQueries(
	orders port.OrderRepository,
	customers port.CustomerRepository,
	couriers port.CourierRepository,
	pickupPoints port.PickupPointRepository,
	now func() time.Time,
) *ReportQueries {
	return &ReportQueries{orders, customers, couriers, pickupPoints, now}
}

// Daily собирает отчёт по заказам, созданным в указанный день.
func (q *ReportQueries) Daily(day time.Time) (DailyReport, error) {
	date := day.Format(time.DateOnly)
	report := DailyReport{Date: date, GeneratedAt: q.now()}

	orders, err := q.orders.List()
	if err != nil {
		return report, err
	}
	refs, err := loadReferences(q.customers, q.couriers, q.pickupPoints)
	if err != nil {
		return report, err
	}

	statusCount := map[domain.OrderStatus]int{}
	deliveryStat := map[domain.DeliveryType]*DeliveryStat{}
	deliveredBy := map[string]int{}

	for _, o := range orders {
		if o.CreatedAt.In(time.Local).Format(time.DateOnly) != date {
			continue
		}
		v := refs.orderView(o)
		report.TotalOrders++
		statusCount[o.Status]++
		switch {
		case o.Status == domain.StatusDelivered:
			report.DeliveredOrders++
			deliveredBy[o.CourierID]++
		case o.Status == domain.StatusCancelled:
			report.CancelledOrders++
		default:
			report.ActiveOrders++
		}
		if o.Status != domain.StatusCancelled {
			report.Revenue += o.Price
			stat := deliveryStat[o.DeliveryType]
			if stat == nil {
				stat = &DeliveryStat{Type: string(o.DeliveryType), Title: v.DeliveryTitle}
				deliveryStat[o.DeliveryType] = stat
			}
			stat.Count++
			stat.Revenue += o.Price
		}
		report.Orders = append(report.Orders, ReportOrderRow{
			ID: o.ID, Created: o.CreatedAt, Customer: v.CustomerName, Delivery: v.DeliveryTitle,
			Status: v.StatusTitle, Courier: v.CourierName, Price: o.Price,
		})
	}

	if paid := report.TotalOrders - report.CancelledOrders; paid > 0 {
		report.AveragePrice = report.Revenue / domain.Money(paid)
	}
	for _, s := range domain.AllStatuses() {
		report.ByStatus = append(report.ByStatus, StatusCount{string(s), s.Title(), statusCount[s]})
	}
	for _, m := range domain.DeliveryMethods() {
		if stat, ok := deliveryStat[m.Type()]; ok {
			report.ByDelivery = append(report.ByDelivery, *stat)
		}
	}
	for _, c := range refs.couriers {
		report.Couriers = append(report.Couriers, CourierLoad{
			ID: c.ID, Name: c.Name, ActiveOrders: c.ActiveOrders, Capacity: c.Capacity, Delivered: deliveredBy[c.ID],
		})
	}
	slices.SortFunc(report.Couriers, func(a, b CourierLoad) int { return compareByName(a.Name, b.Name) })
	slices.SortFunc(report.Orders, func(a, b ReportOrderRow) int { return a.Created.Compare(b.Created) })
	return report, nil
}

// JSON — отчёт в формате JSON.
func (r DailyReport) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// CSV — отчёт в формате CSV (открывается в Excel).
func (r DailyReport) CSV() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // BOM, чтобы Excel правильно показал кириллицу
	w := csv.NewWriter(&buf)
	w.Comma = ';'

	money := func(m domain.Money) string { return strconv.FormatInt(int64(m), 10) }
	rows := [][]string{
		{"Отчёт Parcel Hub за", r.Date},
		{"Всего заказов", strconv.Itoa(r.TotalOrders)},
		{"Активных", strconv.Itoa(r.ActiveOrders)},
		{"Выдано", strconv.Itoa(r.DeliveredOrders)},
		{"Отменено", strconv.Itoa(r.CancelledOrders)},
		{"Выручка, ₸", money(r.Revenue)},
		{"Средний чек, ₸", money(r.AveragePrice)},
		{},
		{"ID", "Создан", "Клиент", "Доставка", "Статус", "Курьер", "Цена, ₸"},
	}
	for _, o := range r.Orders {
		rows = append(rows, []string{
			o.ID, o.Created.In(time.Local).Format("15:04"), o.Customer, o.Delivery, o.Status, o.Courier, money(o.Price),
		})
	}
	rows = append(rows, []string{}, []string{"Курьер", "Активных", "Вместимость", "Выдано за день"})
	for _, c := range r.Couriers {
		rows = append(rows, []string{c.Name, strconv.Itoa(c.ActiveOrders), strconv.Itoa(c.Capacity), strconv.Itoa(c.Delivered)})
	}

	if err := w.WriteAll(rows); err != nil {
		return nil, fmt.Errorf("запись CSV: %w", err)
	}
	return buf.Bytes(), nil
}
