package qdrant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
