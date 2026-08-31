package engine

import (
	"sync"
	"sync/atomic"
	"time"
)

type EventKind string

const (
	EventChangeAccepted         EventKind = "change.accepted"
	EventProposalCreated        EventKind = "proposal.created"
	EventProposalEvaluated      EventKind = "proposal.evaluated"
	EventProposalApproved       EventKind = "proposal.approved"
	EventReleaseStarted         EventKind = "release.started"
	EventReleaseCompleted       EventKind = "release.completed"
	EventReleaseFailed          EventKind = "release.failed"
	EventIntentRecoveryRequired EventKind = "intent.recovery_required"
)

type Event struct {
	Kind      EventKind `json:"kind"`
	LedgerID  string    `json:"ledgerId"`
	SubjectID string    `json:"subjectId"`
	At        time.Time `json:"at"`
}

type Broker struct {
	mu          sync.RWMutex
	subscribers map[uint64]chan Event
	nextID      uint64
	dropped     atomic.Uint64
}

func (broker *Broker) Subscribe(buffer int) (<-chan Event, func()) {
	if buffer < 0 {
		buffer = 0
	}
	channel := make(chan Event, buffer)
	broker.mu.Lock()
	if broker.subscribers == nil {
		broker.subscribers = make(map[uint64]chan Event)
	}
	id := broker.nextID
	broker.nextID++
	broker.subscribers[id] = channel
	broker.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			broker.mu.Lock()
			delete(broker.subscribers, id)
			close(channel)
			broker.mu.Unlock()
		})
	}
	return channel, unsubscribe
}

func (broker *Broker) Publish(event Event) {
	broker.mu.RLock()
	defer broker.mu.RUnlock()
	for _, channel := range broker.subscribers {
		select {
		case channel <- event:
		default:
			broker.dropped.Add(1)
		}
	}
}

func (broker *Broker) Dropped() uint64 {
	return broker.dropped.Load()
}

func (engine *Engine) Events() *Broker {
	return engine.events
}

func (engine *Engine) publish(kind EventKind, ledgerID, subjectID string) {
	engine.events.Publish(Event{Kind: kind, LedgerID: ledgerID, SubjectID: subjectID, At: time.Now().UTC()})
}
