package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"parcelhub/internal/application/events"
	"parcelhub/internal/domain"
)

// EventStream — наблюдатель (Observer), который пересылает доменные события
// в браузер через Server-Sent Events. Интерфейс показывает уведомление
// и сам обновляет таблицы.
type EventStream struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

func NewEventStream() *EventStream {
	return &EventStream{clients: map[chan []byte]struct{}{}}
}

type streamMessage struct {
	Type    string    `json:"type"`
	OrderID string    `json:"orderId"`
	Message string    `json:"message"`
	At      time.Time `json:"at"`
}

// Handle вызывается шиной событий.
func (s *EventStream) Handle(e domain.Event) {
	payload, err := json.Marshal(streamMessage{
		Type: e.EventName(), OrderID: e.AggregateID(), Message: events.Describe(e), At: e.OccurredAt(),
	})
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.clients {
		select {
		case ch <- payload:
		default: // клиент не успевает читать — пропускаем сообщение
		}
	}
}

func (s *EventStream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming не поддерживается", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan []byte, 16)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
	}()

	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
