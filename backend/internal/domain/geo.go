package domain

import "math"

// Location — точка на упрощённой карте города (координаты в км).
type Location struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func (a Location) DistanceTo(b Location) float64 {
	return math.Round(math.Hypot(a.X-b.X, a.Y-b.Y)*10) / 10
}

// District — район города. Используется для расчёта расстояния.
type District struct {
	Code     string   `json:"code"`
	Name     string   `json:"name"`
	Location Location `json:"-"`
}

// HubLocation — сортировочный склад, откуда уезжают все посылки.
var HubLocation = Location{X: 0, Y: 2}

var districts = []District{
	{Code: "almaly", Name: "Алмалинский", Location: Location{0, 0}},
	{Code: "bostandyk", Name: "Бостандыкский", Location: Location{-2, -5}},
	{Code: "medeu", Name: "Медеуский", Location: Location{4, -3}},
	{Code: "auezov", Name: "Ауэзовский", Location: Location{-7, -1}},
	{Code: "alatau", Name: "Алатауский", Location: Location{-9, 6}},
	{Code: "turksib", Name: "Турксибский", Location: Location{3, 8}},
	{Code: "zhetysu", Name: "Жетысуский", Location: Location{-1, 6}},
	{Code: "nauryzbay", Name: "Наурызбайский", Location: Location{-14, -6}},
}

func Districts() []District { return append([]District(nil), districts...) }

func FindDistrict(code string) (District, bool) {
	for _, d := range districts {
		if d.Code == code {
			return d, true
		}
	}
	return District{}, false
}

// DistanceFromHub — расстояние от склада до района (не меньше 1 км).
func DistanceFromHub(districtCode string) (float64, error) {
	d, ok := FindDistrict(districtCode)
	if !ok {
		return 0, RuleViolation("неизвестный район %q", districtCode)
	}
	return max(1, HubLocation.DistanceTo(d.Location)), nil
}

// distanceBetween — расстояние между двумя районами (для выбора курьера).
func distanceBetween(fromCode, toCode string) float64 {
	from, ok1 := FindDistrict(fromCode)
	to, ok2 := FindDistrict(toCode)
	if !ok1 || !ok2 {
		return math.MaxFloat64
	}
	return from.Location.DistanceTo(to.Location)
}
