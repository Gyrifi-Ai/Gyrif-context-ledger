package inference

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
