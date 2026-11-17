package domain

// ПАТТЕРН FACTORY METHOD
//
// Product         — DeliveryMethod (тариф, лимит веса, какие статусы дальше).
// ConcreteProduct — doorDelivery, pickupPointDelivery, lockerDelivery.
// Creator         — DeliveryCreator с фабричным методом CreateMethod().
// ConcreteCreator — DoorCreator, PickupPointCreator, LockerCreator.
//
// Остальной код работает только с интерфейсом DeliveryMethod
// и не знает, какой именно способ доставки внутри.

type DeliveryType string

const (
	DeliveryDoor        DeliveryType = "door"
	DeliveryPickupPoint DeliveryType = "pickup_point"
	DeliveryLocker      DeliveryType = "locker"
)

// Product
type DeliveryMethod interface {
	Type() DeliveryType
	Title() string
	BaseFee() Money
	MaxWeightKg() float64
	NeedsPickupPoint() bool
	NextStatuses(from OrderStatus) []OrderStatus
}

// Creator
type DeliveryCreator interface {
	CreateMethod() DeliveryMethod
}

// Concrete creators
type DoorCreator struct{}
type PickupPointCreator struct{}
type LockerCreator struct{}

func (DoorCreator) CreateMethod() DeliveryMethod        { return doorDelivery{} }
func (PickupPointCreator) CreateMethod() DeliveryMethod { return pickupPointDelivery{} }
func (LockerCreator) CreateMethod() DeliveryMethod      { return lockerDelivery{} }

var creators = map[DeliveryType]DeliveryCreator{
	DeliveryDoor:        DoorCreator{},
	DeliveryPickupPoint: PickupPointCreator{},
	DeliveryLocker:      LockerCreator{},
}

// DeliveryTypes — порядок способов доставки в интерфейсе.
var DeliveryTypes = []DeliveryType{DeliveryDoor, DeliveryPickupPoint, DeliveryLocker}

// NewDeliveryMethod выбирает создателя по типу и вызывает фабричный метод.
func NewDeliveryMethod(t DeliveryType) (DeliveryMethod, error) {
	creator, ok := creators[t]
	if !ok {
		return nil, RuleViolation("неизвестный способ доставки %q", t)
	}
	return creator.CreateMethod(), nil
}

// --- Concrete products ---

// До двери: после «В пути» сразу «Выдан».
var doorFlow = map[OrderStatus][]OrderStatus{
	StatusCreated:   {StatusAssigned, StatusCancelled},
	StatusAssigned:  {StatusInTransit, StatusCancelled},
	StatusInTransit: {StatusDelivered, StatusCancelled},
}

// Через ПВЗ или постамат: посылка сначала ждёт в точке выдачи.
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
