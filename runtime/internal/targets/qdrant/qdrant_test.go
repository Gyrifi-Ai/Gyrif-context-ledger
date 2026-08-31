package qdrant

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets"
)

func TestReadUsesConfiguredCollectionAndAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/collections/context/points/42" {
			t.Errorf("unexpected path %s", request.URL.Path)
		}
		if request.Header.Get("api-key") != "secret" {
			t.Error("API key missing")
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"result":{"id":42,"payload":{"answer":"yes"}}}`))
	}))
	defer server.Close()
	adapter, err := New(server.URL, "context", "secret")
	if err != nil {
		t.Fatal(err)
	}
	value, err := adapter.Read(context.Background(), "42")
	if err != nil {
		t.Fatal(err)
	}
	if !value.Exists || value.Fingerprint == "" {
		t.Fatalf("unexpected value: %#v", value)
	}
}

func TestRequestClassifiesTargetFailures(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   error
	}{
		{name: "not found", status: http.StatusNotFound, want: targets.ErrNotFound},
		{name: "semantic", status: http.StatusBadRequest, want: targets.ErrSemantic},
		{name: "authentication", status: http.StatusUnauthorized, want: targets.ErrAuthentication},
		{name: "unavailable", status: http.StatusServiceUnavailable, want: targets.ErrUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(test.status) }))
			defer server.Close()
			adapter, err := New(server.URL, "context", "never-log-this-key")
			if err != nil {
				t.Fatal(err)
			}
			err = adapter.Health(context.Background())
			if !errors.Is(err, test.want) || strings.Contains(err.Error(), "never-log-this-key") {
				t.Fatalf("classified error = %v", err)
			}
		})
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	adapter, err := New("http://"+address, "context", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.Health(context.Background()); !errors.Is(err, targets.ErrUnavailable) {
		t.Fatalf("transport error = %v", err)
	}
}

func TestEquivalentJSONHandlesCosineNormalization(t *testing.T) {
	expected := []byte(`{"id":42,"vector":[0.1,0.2],"payload":{"answer":"yes"}}`)
	actual := []byte(`{"id":42,"vector":[0.4472136,0.8944272],"payload":{"answer":"yes"}}`)
	if !equivalentJSON(expected, actual, "Cosine") {
		t.Fatal("cosine-normalized vector should be semantically equivalent")
	}
	if equivalentJSON(expected, actual, "Dot") {
		t.Fatal("dot-product vectors must preserve magnitude")
	}
}

func TestVerifyReturnsStructuredSemanticMismatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"result":{"id":42,"vector":[1,0],"payload":{"answer":"foreign"}}}`))
	}))
	defer server.Close()
	adapter, err := New(server.URL, "context", "")
	if err != nil {
		t.Fatal(err)
	}
	err = adapter.Verify(context.Background(), targets.Plan{Operations: []targets.Operation{{Unit: "42", Action: ledger.ChangePut, Desired: json.RawMessage(`{"id":42,"vector":[1,0],"payload":{"answer":"expected"}}`), DesiredFingerprint: "sha256:expected", TargetMetric: "Cosine"}}})
	var verificationError *targets.VerificationError
	if !errors.As(err, &verificationError) || len(verificationError.Mismatches) != 1 || verificationError.Mismatches[0].Unit != "42" || verificationError.Mismatches[0].Expected != "sha256:expected" || verificationError.Mismatches[0].Observed == "" {
		t.Fatalf("verification error = %#v", err)
	}
}
