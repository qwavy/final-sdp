// Package events — ПАТТЕРН OBSERVER.
//
// Subject          — Bus (хранит подписчиков и рассылает им события).
// Observer         — Subscriber.
// ConcreteObserver — ActivityRecorder (журнал), CustomerNotifier (SMS клиенту),
//
//	http.EventStream (живые уведомления в браузере).
//
// Команды только публикуют событие «статус изменился» и не знают, кто
// и как на него отреагирует. Новая реакция = новый подписчик в main.go.
package events

import (
	"slices"
	"sync"

	"parcelhub/internal/domain"
)

// Subscriber — Observer.
type Subscriber interface {
	Handle(e domain.Event)
}

// Bus — Subject.
type Bus struct {
	mu          sync.RWMutex
	subscribers []Subscriber
}

func NewBus() *Bus { return &Bus{} }

func (b *Bus) Subscribe(s Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers = append(b.subscribers, s)
}

// Publish синхронно передаёт каждое событие всем подписчикам.
func (b *Bus) Publish(events ...domain.Event) {
	b.mu.RLock()
	subscribers := slices.Clone(b.subscribers)
	b.mu.RUnlock()

	for _, e := range events {
		for _, s := range subscribers {
			s.Handle(e)
		}
	}
}
