// Package notify — ПАТТЕРН ADAPTER.
//
// Target  — port.Notifier: Notify(Notification) error. Этого интерфейса ждёт приложение.
// Adaptee — SmsGatewayClient: «сторонний SDK» SMS-шлюза со своим неудобным интерфейсом:
//
//	телефон без «+», текст до 160 символов, результат — числовой код вместо error.
//
// Adapter — SmsNotifier: переводит наш вызов в вызов SDK и код ответа в error.
//
// Приложение (CustomerNotifier) ничего не знает о шлюзе. Чтобы перейти на другой
// сервис (email, Telegram), достаточно написать ещё один адаптер под port.Notifier.
package notify

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"parcelhub/internal/application/port"
)

// --- Adaptee: имитация стороннего SDK. Его код мы «не можем менять». ---

const (
	GatewayOK            = 200
	GatewayBadNumber     = 400
	GatewayTooLong       = 413
	GatewayUnavailable   = 503
	gatewayMaxBodyLength = 160
)

// SmsGatewayClient вместо реальной отправки дописывает сообщение в файл-«outbox».
type SmsGatewayClient struct {
	mu         sync.Mutex
	outboxPath string
	counter    int
}

func NewSmsGatewayClient(outboxPath string) *SmsGatewayClient {
	return &SmsGatewayClient{outboxPath: outboxPath}
}

// SendSMS — API шлюза: msisdn — только цифры (77011234567), body — до 160 символов.
func (g *SmsGatewayClient) SendSMS(msisdn, sender, body string) (statusCode int, messageID string) {
	if len(msisdn) != 11 || strings.Trim(msisdn, "0123456789") != "" {
		return GatewayBadNumber, ""
	}
	if utf8.RuneCountInString(body) > gatewayMaxBodyLength {
		return GatewayTooLong, ""
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	g.counter++
	messageID = fmt.Sprintf("sms-%d-%d", time.Now().Unix(), g.counter)

	line, _ := json.Marshal(map[string]string{
		"id": messageID, "to": msisdn, "from": sender, "body": body, "sentAt": time.Now().Format(time.RFC3339),
	})
	f, err := os.OpenFile(g.outboxPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return GatewayUnavailable, ""
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return GatewayUnavailable, ""
	}
	return GatewayOK, messageID
}

// --- Adapter ---

// SmsNotifier реализует port.Notifier поверх SmsGatewayClient.
type SmsNotifier struct {
	gateway *SmsGatewayClient
	sender  string
}

func NewSmsNotifier(gateway *SmsGatewayClient, sender string) *SmsNotifier {
	return &SmsNotifier{gateway: gateway, sender: sender}
}

var _ port.Notifier = (*SmsNotifier)(nil)

func (a *SmsNotifier) Notify(n port.Notification) error {
	msisdn := strings.TrimPrefix(n.Phone, "+") // +77011234567 → 77011234567
	body := truncate(n.Text, gatewayMaxBodyLength)

	code, _ := a.gateway.SendSMS(msisdn, a.sender, body)
	switch code {
	case GatewayOK:
		return nil
	case GatewayBadNumber:
		return fmt.Errorf("шлюз не принял номер %s", n.Phone)
	case GatewayTooLong:
		return fmt.Errorf("сообщение слишком длинное")
	default:
		return fmt.Errorf("SMS-шлюз недоступен (код %d)", code)
	}
}

func truncate(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes-1]) + "…"
}
