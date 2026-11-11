package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Money — сумма в тенге (целое число).
type Money int64

// Percent возвращает p процентов от суммы с округлением.
func (m Money) Percent(p float64) Money {
	return Money(math.Round(float64(m) * p / 100))
}

func (m Money) String() string { return fmt.Sprintf("%d ₸", int64(m)) }

// NewID создаёт короткий читаемый идентификатор, например ORD-3F9A1C.
func NewID(prefix string) string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return prefix + "-" + strings.ToUpper(hex.EncodeToString(b))
}

var phonePattern = regexp.MustCompile(`^\+7\d{10}$`)

// NormalizePhone приводит "8 (701) 123-45-67" к виду "+77011234567".
func NormalizePhone(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r == '+' || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	p := b.String()
	if len(p) == 11 && p[0] == '8' {
		p = "+7" + p[1:]
	}
	if len(p) == 11 && p[0] == '7' {
		p = "+" + p
	}
	return p
}

// validateContact проверяет имя и телефон (общая логика клиента и курьера).
func validateContact(errs *ValidationError, name, phone string) {
	switch {
	case name == "":
		errs.Add("name", "укажите имя")
	case utf8.RuneCountInString(name) > 80:
		errs.Add("name", "имя не длиннее 80 символов")
	}
	switch {
	case phone == "":
		errs.Add("phone", "укажите телефон")
	case !phonePattern.MatchString(phone):
		errs.Add("phone", "телефон в формате +7XXXXXXXXXX")
	}
}
