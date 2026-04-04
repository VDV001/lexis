package eventbus

import (
	"log"
	"sync"
)

const (
	EventRoundCompleted  = "round.completed"
	EventWordsDiscovered = "words.discovered"
)

type Event struct {
	Type    string
	Payload any
}

type RoundCompletedPayload struct {
	UserID        string
	SessionID     string
	Mode          string
	IsCorrect     bool
	ErrorType     string
	Question      string
	UserAnswer    string
	CorrectAnswer string
	Explanation   string
}

type WordsDiscoveredPayload struct {
	UserID   string
	Language string
	Words    []string
	Context  string
}

type Publisher interface {
	Publish(event Event)
}

type handler func(Event)

type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]handler
}

func New() *Bus {
	return &Bus{
		handlers: make(map[string][]handler),
	}
}

func (b *Bus) Subscribe(eventType string, h func(Event)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], h)
}

func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	handlers := make([]handler, len(b.handlers[event.Type]))
	copy(handlers, b.handlers[event.Type])
	b.mu.RUnlock()

	for _, h := range handlers {
		go func(fn handler) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("eventbus: handler panic: %v", r)
				}
			}()
			fn(event)
		}(h)
	}
}
