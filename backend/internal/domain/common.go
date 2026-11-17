package domain

import (
	"crypto/rand"
	"encoding/hex"
	"math"
	"regexp"
	"strings"
	"unicode"
)

// Money — сумма в тенге.
type Money int64

// Percent — p процентов от суммы.
func (m Money) Percent(p float64) Money {
	return Money(math.Round(float64(m) * p / 100))
}

// NewID создаёт короткий ID, например ORD-3F9A1C.
func NewID(prefix string) string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return prefix + "-" + strings.ToUpper(hex.EncodeToString(b))
}

var phonePattern = regexp.MustCompile(`^\+7\d{10}$`)

// NormalizePhone: "8 (701) 123-45-67" → "+77011234567".
func NormalizePhone(raw string) string {
	digits := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, raw)
	if len(digits) == 11 && digits[0] == '8' {
		digits = "7" + digits[1:]
	}
	return "+" + digits
}

// checkContact проверяет имя и телефон (общая проверка клиента и курьера).
func checkContact(errs ValidationError, name, phone string) {
	if name == "" {
		errs["name"] = "укажите имя"
	}
	if !phonePattern.MatchString(phone) {
		errs["phone"] = "телефон в формате +7XXXXXXXXXX"
	}
}
