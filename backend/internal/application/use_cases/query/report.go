package query

import (
	"fmt"
	"strings"
	"time"

	"parcelhub/internal/domain"
)

// DailyReport — итог работы за день.
type DailyReport struct {
	Date            string        `json:"date"`
	TotalOrders     int           `json:"totalOrders"`
	ActiveOrders    int           `json:"activeOrders"`
	DeliveredOrders int           `json:"deliveredOrders"`
	CancelledOrders int           `json:"cancelledOrders"`
	Revenue         domain.Money  `json:"revenue"`
	AveragePrice    domain.Money  `json:"averagePrice"`
	Couriers        []CourierLoad `json:"couriers"`
	Orders          []ReportRow   `json:"orders"`
}

type CourierLoad struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ActiveOrders int    `json:"activeOrders"`
	Capacity     int    `json:"capacity"`
	Delivered    int    `json:"delivered"`
}

type ReportRow struct {
	ID       string       `json:"id"`
	Created  time.Time    `json:"created"`
	Customer string       `json:"customer"`
	Delivery string       `json:"delivery"`
	Status   string       `json:"status"`
	Courier  string       `json:"courier"`
	Price    domain.Money `json:"price"`
}

// DailyReport собирает отчёт по заказам, созданным в этот день.
func (q *OrderQueries) DailyReport(day time.Time) (DailyReport, error) {
	r := DailyReport{Date: day.Format(time.DateOnly), Couriers: []CourierLoad{}, Orders: []ReportRow{}}

	orders, err := q.List("", "")
	if err != nil {
		return r, err
	}
	deliveredBy := map[string]int{}
	for _, o := range orders {
		if o.CreatedAt.In(time.Local).Format(time.DateOnly) != r.Date {
			continue
		}
		r.TotalOrders++
		switch o.Status {
		case domain.StatusDelivered:
			r.DeliveredOrders++
			deliveredBy[o.CourierID]++
		case domain.StatusCancelled:
			r.CancelledOrders++
		default:
			r.ActiveOrders++
		}
		if o.Status != domain.StatusCancelled {
			r.Revenue += o.Price
		}
		r.Orders = append(r.Orders, ReportRow{o.ID, o.CreatedAt, o.CustomerName, o.DeliveryTitle, o.StatusTitle, o.CourierName, o.Price})
	}
	if paid := r.TotalOrders - r.CancelledOrders; paid > 0 {
		r.AveragePrice = r.Revenue / domain.Money(paid)
	}

	couriers, err := q.couriers.List()
	if err != nil {
		return r, err
	}
	for _, c := range couriers {
		r.Couriers = append(r.Couriers, CourierLoad{c.ID, c.Name, c.ActiveOrders, c.Capacity, deliveredBy[c.ID]})
	}
	return r, nil
}

// CSV — отчёт для Excel (разделитель «;»).
func (r DailyReport) CSV() []byte {
	var b strings.Builder
	b.WriteString("\xEF\xBB\xBF") // чтобы Excel правильно показал кириллицу
	fmt.Fprintf(&b, "Отчёт Parcel Hub за;%s\n", r.Date)
	fmt.Fprintf(&b, "Всего заказов;%d\nВ работе;%d\nВыдано;%d\nОтменено;%d\n",
		r.TotalOrders, r.ActiveOrders, r.DeliveredOrders, r.CancelledOrders)
	fmt.Fprintf(&b, "Выручка, ₸;%d\nСредний чек, ₸;%d\n\n", r.Revenue, r.AveragePrice)

	b.WriteString("ID;Время;Клиент;Доставка;Статус;Курьер;Цена, ₸\n")
	for _, o := range r.Orders {
		fmt.Fprintf(&b, "%s;%s;%s;%s;%s;%s;%d\n",
			o.ID, o.Created.In(time.Local).Format("15:04"), o.Customer, o.Delivery, o.Status, o.Courier, o.Price)
	}
	return []byte(b.String())
}
