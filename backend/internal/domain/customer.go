package domain

import (
	"strings"
	"time"
)

// Customer — клиент, которому везут посылку.
type Customer struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewCustomer(name, phone string, now time.Time) (*Customer, error) {
	c := &Customer{ID: NewID("CUS"), CreatedAt: now}
	if err := c.Update(name, phone); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Customer) Update(name, phone string) error {
	name = strings.TrimSpace(name)
	phone = NormalizePhone(phone)

	errs := ValidationError{}
	checkContact(errs, name, phone)
	if err := errs.OrNil(); err != nil {
		return err
	}
	c.Name, c.Phone = name, phone
	return nil
}
