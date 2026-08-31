package engine

import (
	"sync"
	"testing"
	"time"
)

func TestBrokerSubscribePublishAndIdempotentUnsubscribe(t *testing.T) {
	broker := &Broker{}
	events, unsubscribe := broker.Subscribe(1)
	want := Event{Kind: EventChangeAccepted, LedgerID: "ldg_one", SubjectID: "chg_one", At: time.Now().UTC()}

	broker.Publish(want)
	select {
	case got := <-events:
		if got != want {
			t.Fatalf("event mismatch: got %#v want %#v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("event was not delivered")
	}

	unsubscribe()
	unsubscribe()
	if _, open := <-events; open {
		t.Fatal("subscription channel remained open")
	}
	broker.Publish(want)
}

func TestBrokerPublishDoesNotBlockSlowSubscriber(t *testing.T) {
	broker := &Broker{}
	_, unsubscribe := broker.Subscribe(1)
	defer unsubscribe()
	broker.Publish(Event{Kind: EventChangeAccepted})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 100 {
			broker.Publish(Event{Kind: EventProposalCreated})
		}
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked on a full subscriber buffer")
	}
	if got := broker.Dropped(); got != 100 {
		t.Fatalf("dropped count = %d, want 100", got)
	}
}

func TestBrokerPublishAndUnsubscribeAreSafeConcurrently(t *testing.T) {
	broker := &Broker{}
	for range 100 {
		_, unsubscribe := broker.Subscribe(1)
		var wait sync.WaitGroup
		wait.Add(2)
		go func() {
			defer wait.Done()
			for range 100 {
				broker.Publish(Event{Kind: EventProposalEvaluated})
			}
		}()
		go func() {
			defer wait.Done()
			unsubscribe()
			unsubscribe()
		}()
		wait.Wait()
	}
}
