package domain

// ПАТТЕРН FACTORY METHOD
//
// Product         — DeliveryMethod (способ доставки: тариф, лимит веса, правила статусов).
// ConcreteProduct — doorDelivery, pickupPointDelivery, lockerDelivery.
// Creator         — DeliveryCreator с фабричным методом CreateMethod().
// ConcreteCreator — DoorDeliveryCreator, PickupPointDeliveryCreator, LockerDeliveryCreator.
//
// Остальной код (Order, OrderBuilder, расчёт цены) работает только с интерфейсом
// DeliveryMethod и не знает конкретных типов. Новый способ доставки =
// новый продукт + новый создатель + одна строка в deliveryCreators.

type DeliveryType string

const (
	DeliveryDoor        DeliveryType = "door"
	DeliveryPickupPoint DeliveryType = "pickup_point"
	DeliveryLocker      DeliveryType = "locker"
)

// DeliveryMethod — Product.
type DeliveryMethod interface {
	Type() DeliveryType
	Title() string
	BaseFee() Money
	MaxWeightKg() float64
	NeedsPickupPoint() bool
	NextStatuses(from OrderStatus) []OrderStatus
}

// DeliveryCreator — Creator.
type DeliveryCreator interface {
	CreateMethod() DeliveryMethod
}

type DoorDeliveryCreator struct{}
type PickupPointDeliveryCreator struct{}
type LockerDeliveryCreator struct{}

func (DoorDeliveryCreator) CreateMethod() DeliveryMethod        { return doorDelivery{} }
func (PickupPointDeliveryCreator) CreateMethod() DeliveryMethod { return pickupPointDelivery{} }
func (LockerDeliveryCreator) CreateMethod() DeliveryMethod      { return lockerDelivery{} }

// Порядок важен: в таком порядке способы показываются в интерфейсе.
var deliveryTypes = []DeliveryType{DeliveryDoor, DeliveryPickupPoint, DeliveryLocker}

var deliveryCreators = map[DeliveryType]DeliveryCreator{
	DeliveryDoor:        DoorDeliveryCreator{},
	DeliveryPickupPoint: PickupPointDeliveryCreator{},
	DeliveryLocker:      LockerDeliveryCreator{},
}

// NewDeliveryMethod находит создателя по типу и вызывает фабричный метод.
func NewDeliveryMethod(t DeliveryType) (DeliveryMethod, error) {
	creator, ok := deliveryCreators[t]
	if !ok {
		return nil, RuleViolation("неизвестный способ доставки %q", t)
	}
	return creator.CreateMethod(), nil
}

// DeliveryMethods — все способы доставки (для формы в интерфейсе).
func DeliveryMethods() []DeliveryMethod {
	methods := make([]DeliveryMethod, 0, len(deliveryTypes))
	for _, t := range deliveryTypes {
		methods = append(methods, deliveryCreators[t].CreateMethod())
	}
	return methods
}

// --- Concrete products ---

// Курьер везёт до двери: после «В пути» сразу «Выдан».
var doorFlow = map[OrderStatus][]OrderStatus{
	StatusCreated:   {StatusAssigned, StatusCancelled},
	StatusAssigned:  {StatusInTransit, StatusCancelled},
	StatusInTransit: {StatusDelivered, StatusCancelled},
}

// Через ПВЗ/постамат: посылка сначала ждёт в точке выдачи.
var pickupFlow = map[OrderStatus][]OrderStatus{
	StatusCreated:       {StatusAssigned, StatusCancelled},
	StatusAssigned:      {StatusInTransit, StatusCancelled},
	StatusInTransit:     {StatusAtPickupPoint},
	StatusAtPickupPoint: {StatusDelivered, StatusCancelled},
}

type doorDelivery struct{}

func (doorDelivery) Type() DeliveryType                          { return DeliveryDoor }
func (doorDelivery) Title() string                               { return "Курьером до двери" }
func (doorDelivery) BaseFee() Money                              { return 1000 }
func (doorDelivery) MaxWeightKg() float64                        { return 50 }
func (doorDelivery) NeedsPickupPoint() bool                      { return false }
func (doorDelivery) NextStatuses(from OrderStatus) []OrderStatus { return doorFlow[from] }

type pickupPointDelivery struct{}

func (pickupPointDelivery) Type() DeliveryType                          { return DeliveryPickupPoint }
func (pickupPointDelivery) Title() string                               { return "В пункт выдачи" }
func (pickupPointDelivery) BaseFee() Money                              { return 500 }
func (pickupPointDelivery) MaxWeightKg() float64                        { return 30 }
func (pickupPointDelivery) NeedsPickupPoint() bool                      { return true }
func (pickupPointDelivery) NextStatuses(from OrderStatus) []OrderStatus { return pickupFlow[from] }

type lockerDelivery struct{}

func (lockerDelivery) Type() DeliveryType                          { return DeliveryLocker }
func (lockerDelivery) Title() string                               { return "В постамат" }
func (lockerDelivery) BaseFee() Money                              { return 300 }
func (lockerDelivery) MaxWeightKg() float64                        { return 15 }
func (lockerDelivery) NeedsPickupPoint() bool                      { return true }
func (lockerDelivery) NextStatuses(from OrderStatus) []OrderStatus { return pickupFlow[from] }
