// Package notify — ПАТТЕРН ADAPTER.
//
// Target  — port.Notifier: Notify(Notification) error. Его ждёт приложение.
// Adaptee — SmsGateway: «чужой» SMS-сервис со своим неудобным интерфейсом:
//
//	номер без «+», текст до 160 символов, результат — код (200, 400 …), а не error.
//
// Adapter — SmsNotifier: переводит наш вызов в вызов шлюза, а код — в error.
package notify

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"parcelhub/internal/application/port"
)

// --- Adaptee: «чужой» сервис, его код мы не меняем ---

// SmsGateway вместо настоящей отправки дописывает SMS в файл.
type SmsGateway struct {
	OutboxFile string
}

// Send: number — только цифры (77011234567), text — до 160 символов.
// Возвращает 200 — отправлено, 400 — плохой номер, 413 — длинный текст, 503 — ошибка.
func (g *SmsGateway) Send(number, text string) int {
	if len(number) != 11 || strings.Trim(number, "0123456789") != "" {
		return 400
	}
	if utf8.RuneCountInString(text) > 160 {
		return 413
	}
	line := fmt.Sprintf("%s  to=%s  %s\n", time.Now().Format(time.DateTime), number, text)
	f, err := os.OpenFile(g.OutboxFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return 503
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		return 503
	}
	return 200
}

// --- Adapter ---

// SmsNotifier реализует port.Notifier поверх SmsGateway.
type SmsNotifier struct {
	gateway *SmsGateway
}

func NewSmsNotifier(gateway *SmsGateway) *SmsNotifier {
	return &SmsNotifier{gateway: gateway}
}

func (a *SmsNotifier) Notify(n port.Notification) error {
	number := strings.TrimPrefix(n.Phone, "+") // +77011234567 → 77011234567
	text := n.Text
	if runes := []rune(text); len(runes) > 160 {
		text = string(runes[:160]) // обрезаем под лимит шлюза
	}

	switch a.gateway.Send(number, text) {
	case 200:
		return nil
	case 400:
		return fmt.Errorf("шлюз не принял номер %s", n.Phone)
	default:
		return fmt.Errorf("SMS-шлюз недоступен")
	}
}
