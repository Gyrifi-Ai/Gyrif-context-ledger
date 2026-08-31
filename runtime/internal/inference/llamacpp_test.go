package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLlamaServerDrainBoundsOutput(t *testing.T) {
	var logs bytes.Buffer
	server := &LlamaServer{logger: slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})), options: supervisorOptions{lineLimit: 4, stderrLines: 2}}
	server.drain(strings.NewReader("123456\nsecond\nthird\n"), true)
	lines := server.stderrSnapshot()
	if len(lines) != 2 || lines[0] != "seco [truncated]" || lines[1] != "thir [truncated]" {
		t.Fatalf("bounded stderr = %#v", lines)
	}
	if output := logs.String(); !strings.Contains(output, "component=llama-server") || !strings.Contains(output, "stream=stderr") || !strings.Contains(output, "[truncated]") {
		t.Fatalf("structured output = %q", output)
	}
}

func TestLlamaServerStopDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	server := &LlamaServer{ctx: ctx, cancel: cancel, logger: slog.New(slog.NewTextHandler(io.Discard, nil)), options: supervisorOptions{backoffBase: time.Hour, backoffCap: time.Hour, stopTimeout: 100 * time.Millisecond}, state: "restarting", done: make(chan struct{})}
	go func() {
		defer close(server.done)
		if server.waitBackoff(time.Hour) {
			t.Error("backoff completed after cancellation")
		}
	}()
	started := time.Now()
	if err := server.Stop(); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Stop took %s", elapsed)
	}
	if server.State() != "stopped" {
		t.Fatalf("state = %q", server.State())
	}
}

func TestLlamaServerRestartLimitSetsFailed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	process := &managedProcess{exit: make(chan error, 1)}
	process.exit <- errors.New("crashed")
	close(process.exit)
	server := &LlamaServer{ctx: ctx, cancel: cancel, logger: slog.New(slog.NewTextHandler(io.Discard, nil)), maxRestarts: 0, options: defaultSupervisorOptions, state: "ready", done: make(chan struct{})}
	go server.supervise(process)
	select {
	case <-server.done:
	case <-time.After(time.Second):
		t.Fatal("supervisor did not finish")
	}
	if server.State() != "failed" || server.Healthy() {
		t.Fatalf("state = %q healthy = %v", server.State(), server.Healthy())
	}
}

func TestLlamaServerStderrSuffix(t *testing.T) {
	server := &LlamaServer{logger: slog.New(slog.NewTextHandler(io.Discard, nil)), options: supervisorOptions{lineLimit: 8, stderrLines: 2}}
	server.drain(bytes.NewBufferString("first\nsecond\nthird\n"), true)
	if suffix := server.stderrSuffix(); suffix != "; stderr: second | third" {
		t.Fatalf("suffix = %q", suffix)
	}
}

func TestLlamaCppProviderParsesStructuredEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/chat/completions" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"model":"gemma-test","choices":[{"message":{"content":"{\"passed\":true,\"summary\":\"consistent\",\"findings\":[]}"}}]}`))
	}))
	defer server.Close()
	provider := NewLlamaCppProvider(server.URL, "gemma.gguf")
	result, err := provider.Evaluate(context.Background(), EvaluationRequest{ProposalHash: "sha256:test", Context: json.RawMessage(`{"unit":"42"}`), Criteria: "consistent"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || result.Model != "gemma-test" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
func TestLlamaCppProviderRejectsFreeFormOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"model":"gemma-test","choices":[{"message":{"content":"looks fine"}}]}`))
	}))
	defer server.Close()
	provider := NewLlamaCppProvider(server.URL, "gemma.gguf")
	if _, err := provider.Evaluate(context.Background(), EvaluationRequest{ProposalHash: "hash"}); err == nil {
		t.Fatal("free-form output must not become governance evidence")
	}
}
