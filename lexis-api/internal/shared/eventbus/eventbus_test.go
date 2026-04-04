package eventbus_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexis-app/lexis-api/internal/shared/eventbus"
)

func TestPublishDeliversToSubscriber(t *testing.T) {
	bus := eventbus.New()

	var received eventbus.Event
	done := make(chan struct{})

	bus.Subscribe("test.event", func(e eventbus.Event) {
		received = e
		close(done)
	})

	bus.Publish(eventbus.Event{Type: "test.event", Payload: "hello"})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	assert.Equal(t, "test.event", received.Type)
	assert.Equal(t, "hello", received.Payload)
}

func TestPublishFansOutToMultipleSubscribers(t *testing.T) {
	bus := eventbus.New()

	var count atomic.Int32
	var wg sync.WaitGroup

	for range 3 {
		wg.Add(1)
		bus.Subscribe("fanout", func(e eventbus.Event) {
			count.Add(1)
			wg.Done()
		})
	}

	bus.Publish(eventbus.Event{Type: "fanout"})

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for all subscribers")
	}

	assert.Equal(t, int32(3), count.Load())
}

func TestPublishNoSubscribersDoesNotPanic(t *testing.T) {
	bus := eventbus.New()
	assert.NotPanics(t, func() {
		bus.Publish(eventbus.Event{Type: "no.listeners"})
	})
}

func TestSubscribersReceiveCorrectPayloadType(t *testing.T) {
	bus := eventbus.New()

	done := make(chan struct{})
	var payload eventbus.RoundCompletedPayload

	bus.Subscribe(eventbus.EventRoundCompleted, func(e eventbus.Event) {
		p, ok := e.Payload.(eventbus.RoundCompletedPayload)
		require.True(t, ok)
		payload = p
		close(done)
	})

	bus.Publish(eventbus.Event{
		Type: eventbus.EventRoundCompleted,
		Payload: eventbus.RoundCompletedPayload{
			UserID:    "u1",
			SessionID: "s1",
			Mode:      "quiz",
			IsCorrect: true,
		},
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	assert.Equal(t, "u1", payload.UserID)
	assert.True(t, payload.IsCorrect)
}

func TestDifferentEventTypesAreIsolated(t *testing.T) {
	bus := eventbus.New()

	var called atomic.Bool
	bus.Subscribe("type.a", func(e eventbus.Event) {
		called.Store(true)
	})

	bus.Publish(eventbus.Event{Type: "type.b"})

	time.Sleep(50 * time.Millisecond)
	assert.False(t, called.Load())
}

func TestBusImplementsPublisher(t *testing.T) {
	var _ eventbus.Publisher = eventbus.New()
}

func TestConcurrentSubscribeAndPublish(t *testing.T) {
	bus := eventbus.New()
	var count atomic.Int32

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Subscribe("concurrent", func(e eventbus.Event) {
				count.Add(1)
			})
		}()
	}
	wg.Wait()

	var publishWg sync.WaitGroup
	for range 5 {
		publishWg.Add(1)
		go func() {
			defer publishWg.Done()
			bus.Publish(eventbus.Event{Type: "concurrent"})
		}()
	}
	publishWg.Wait()

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(50), count.Load()) // 10 subscribers * 5 publishes
}
