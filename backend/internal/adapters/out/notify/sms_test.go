package notify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"parcelhub/internal/application/port"
)

// Adapter: наш формат (+7..., любой длины текст) приводится к формату шлюза.
func TestSmsNotifierAdaptsToGateway(t *testing.T) {
	outbox := filepath.Join(t.TempDir(), "outbox.jsonl")
	adapter := NewSmsNotifier(NewSmsGatewayClient(outbox), "ParcelHub")

	long := strings.Repeat("а", 300)
	if err := adapter.Notify(port.Notification{Phone: "+77011234567", Text: long}); err != nil {
		t.Fatalf("ожидали успешную отправку: %v", err)
	}
	data, _ := os.ReadFile(outbox)
	if !strings.Contains(string(data), `"to":"77011234567"`) {
		t.Fatalf("номер должен уйти без «+»: %s", data)
	}

	if err := adapter.Notify(port.Notification{Phone: "123", Text: "x"}); err == nil {
		t.Fatal("код ошибки шлюза должен превратиться в error")
	}
}
