package domain

import "math"

// District — район города с координатами на упрощённой карте (в км).
type District struct {
	Code string
	Name string
	X, Y float64
}

var Districts = []District{
	{"almaly", "Алмалинский", 0, 0},
	{"bostandyk", "Бостандыкский", -2, -5},
	{"medeu", "Медеуский", 4, -3},
	{"auezov", "Ауэзовский", -7, -1},
	{"alatau", "Алатауский", -9, 6},
	{"turksib", "Турксибский", 3, 8},
	{"zhetysu", "Жетысуский", -1, 6},
	{"nauryzbay", "Наурызбайский", -14, -6},
}

// Склад, откуда уезжают все посылки.
const hubX, hubY = 0.0, 2.0

func FindDistrict(code string) (District, bool) {
	for _, d := range Districts {
		if d.Code == code {
			return d, true
		}
	}
	return District{}, false
}

// DistanceFromHub — расстояние от склада до района (не меньше 1 км).
func DistanceFromHub(code string) (float64, bool) {
	d, ok := FindDistrict(code)
	if !ok {
		return 0, false
	}
	return max(1, distance(hubX, hubY, d.X, d.Y)), true
}

// DistanceBetween — расстояние между двумя районами.
func DistanceBetween(fromCode, toCode string) float64 {
	from, ok1 := FindDistrict(fromCode)
	to, ok2 := FindDistrict(toCode)
	if !ok1 || !ok2 {
		return math.MaxFloat64
	}
	return distance(from.X, from.Y, to.X, to.Y)
}

func distance(x1, y1, x2, y2 float64) float64 {
	return math.Round(math.Hypot(x1-x2, y1-y2)*10) / 10
}
