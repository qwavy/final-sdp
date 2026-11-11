package domain

import (
	"fmt"
	"strings"
	"time"
)

const MaxCourierCapacity = 10

// Courier — курьер. Capacity — сколько заказов он может везти одновременно.
type Courier struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Phone        string    `json:"phone"`
	District     string    `json:"district"`
	Capacity     int       `json:"capacity"`
	ActiveOrders int       `json:"activeOrders"`
	CreatedAt    time.Time `json:"createdAt"`
}

func NewCourier(name, phone, district string, capacity int, now time.Time) (*Courier, error) {
	c := &Courier{ID: NewID("CUR"), CreatedAt: now}
	if err := c.Update(name, phone, district, capacity); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Courier) Update(name, phone, district string, capacity int) error {
	name = strings.TrimSpace(name)
	phone = NormalizePhone(phone)

	var errs ValidationError
	validateContact(&errs, name, phone)
	if _, ok := FindDistrict(district); !ok {
		errs.Add("district", "выберите район")
	}
	switch {
	case capacity < 1 || capacity > MaxCourierCapacity:
		errs.Add("capacity", fmt.Sprintf("вместимость от 1 до %d заказов", MaxCourierCapacity))
	case capacity < c.ActiveOrders:
		errs.Add("capacity", fmt.Sprintf("не меньше текущих заказов (%d)", c.ActiveOrders))
	}
	if err := errs.OrNil(); err != nil {
		return err
	}
	c.Name, c.Phone, c.District, c.Capacity = name, phone, district, capacity
	return nil
}

func (c *Courier) HasCapacity() bool { return c.ActiveOrders < c.Capacity }

func (c *Courier) TakeOrder() error {
	if !c.HasCapacity() {
		return RuleViolation("курьер %s уже везёт максимум заказов (%d)", c.Name, c.Capacity)
	}
	c.ActiveOrders++
	return nil
}

func (c *Courier) ReleaseOrder() {
	if c.ActiveOrders > 0 {
		c.ActiveOrders--
	}
}
