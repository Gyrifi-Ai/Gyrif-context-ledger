package httpinterface

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
)

type streamRecorder struct {
	header  http.Header
	mu      sync.Mutex
	body    bytes.Buffer
	flushes chan string
}

func newStreamRecorder() *streamRecorder {
	return &streamRecorder{header: make(http.Header), flushes: make(chan string, 4)}
}

func (recorder *streamRecorder) Header() http.Header { return recorder.header }
func (recorder *streamRecorder) WriteHeader(_ int)   {}
func (recorder *streamRecorder) Write(value []byte) (int, error) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return recorder.body.Write(value)
}
func (recorder *streamRecorder) Flush() {
	recorder.mu.Lock()
	value := recorder.body.String()
	recorder.mu.Unlock()
	recorder.flushes <- value
}

func receiveFlush(t *testing.T, recorder *streamRecorder) string {
	t.Helper()
	select {
	case value := <-recorder.flushes:
		return value
	case <-time.After(time.Second):
		t.Fatal("event stream did not flush")
		return ""
	}
}

func TestEventsConnectedFrameForwardingFilteringAndCancellation(t *testing.T) {
	application := engine.New(nil, nil, nil)
	server := New(application, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/events/v1?ledgerId=ldg_one", nil).WithContext(ctx)
	recorder := newStreamRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		server.events(recorder, request)
	}()

	connected := receiveFlush(t, recorder)
	if connected != "event: ledger\ndata: {\"status\":\"connected\"}\n\n" {
		t.Fatalf("connected frame = %q", connected)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q", got)
	}

	application.Events().Publish(engine.Event{Kind: engine.EventChangeAccepted, LedgerID: "ldg_other", SubjectID: "chg_other", At: time.Date(2026, 8, 31, 7, 0, 0, 0, time.UTC)})
	application.Events().Publish(engine.Event{Kind: engine.EventProposalCreated, LedgerID: "ldg_one", SubjectID: "pr_one", At: time.Date(2026, 8, 31, 7, 1, 0, 0, time.UTC)})
	forwarded := receiveFlush(t, recorder)
	if strings.Contains(forwarded, "ldg_other") {
		t.Fatal("stream forwarded an event for a different ledger")
	}
	if !strings.Contains(forwarded, "event: proposal.created\n") || !strings.Contains(forwarded, `"ledgerId":"ldg_one"`) || !strings.Contains(forwarded, `"subjectId":"pr_one"`) {
		t.Fatalf("forwarded frame = %q", forwarded)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("event handler did not exit after context cancellation")
	}
	application.Events().Publish(engine.Event{Kind: engine.EventReleaseCompleted})
}
