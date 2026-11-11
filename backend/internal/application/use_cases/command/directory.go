package command

import (
	"time"

	"parcelhub/internal/application/port"
	"parcelhub/internal/domain"
)

type CustomerInput struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type CourierInput struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	District string `json:"district"`
	Capacity int    `json:"capacity"`
}

// CustomerCommands — добавление, изменение и удаление клиентов.
type CustomerCommands struct {
	customers port.CustomerRepository
	orders    port.OrderRepository
	now       func() time.Time
}

func NewCustomerCommands(customers port.CustomerRepository, orders port.OrderRepository, now func() time.Time) *CustomerCommands {
	return &CustomerCommands{customers, orders, now}
}

func (c *CustomerCommands) Create(in CustomerInput) (string, error) {
	customer, err := domain.NewCustomer(in.Name, in.Phone, c.now())
	if err != nil {
		return "", err
	}
	if err := c.ensureUniquePhone(customer); err != nil {
		return "", err
	}
	return customer.ID, c.customers.Save(customer)
}

func (c *CustomerCommands) Update(id string, in CustomerInput) error {
	customer, err := c.customers.Get(id)
	if err != nil {
		return err
	}
	if err := customer.Update(in.Name, in.Phone); err != nil {
		return err
	}
	if err := c.ensureUniquePhone(customer); err != nil {
		return err
	}
	return c.customers.Save(customer)
}

func (c *CustomerCommands) Delete(id string) error {
	if _, err := c.customers.Get(id); err != nil {
		return err
	}
	orders, err := c.orders.List()
	if err != nil {
		return err
	}
	for _, o := range orders {
		if o.CustomerID == id && !o.IsFinal() {
			return domain.RuleViolation("у клиента есть незавершённые заказы (например, %s)", o.ID)
		}
	}
	return c.customers.Delete(id)
}

func (c *CustomerCommands) ensureUniquePhone(customer *domain.Customer) error {
	all, err := c.customers.List()
	if err != nil {
		return err
	}
	for _, other := range all {
		if other.ID != customer.ID && other.Phone == customer.Phone {
			err := &domain.ValidationError{}
			err.Add("phone", "клиент с таким телефоном уже есть: "+other.Name)
			return err
		}
	}
	return nil
}

// CourierCommands — добавление, изменение и удаление курьеров.
type CourierCommands struct {
	couriers port.CourierRepository
	now      func() time.Time
}

func NewCourierCommands(couriers port.CourierRepository, now func() time.Time) *CourierCommands {
	return &CourierCommands{couriers, now}
}

func (c *CourierCommands) Create(in CourierInput) (string, error) {
	courier, err := domain.NewCourier(in.Name, in.Phone, in.District, in.Capacity, c.now())
	if err != nil {
		return "", err
	}
	return courier.ID, c.couriers.Save(courier)
}

func (c *CourierCommands) Update(id string, in CourierInput) error {
	courier, err := c.couriers.Get(id)
	if err != nil {
		return err
	}
	if err := courier.Update(in.Name, in.Phone, in.District, in.Capacity); err != nil {
		return err
	}
	return c.couriers.Save(courier)
}

func (c *CourierCommands) Delete(id string) error {
	courier, err := c.couriers.Get(id)
	if err != nil {
		return err
	}
	if courier.ActiveOrders > 0 {
		return domain.RuleViolation("у курьера %s есть активные заказы (%d)", courier.Name, courier.ActiveOrders)
	}
	return c.couriers.Delete(id)
}
