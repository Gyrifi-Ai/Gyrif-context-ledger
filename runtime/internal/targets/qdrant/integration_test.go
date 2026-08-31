//go:build integration

package qdrant

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets"
)

var (
	integrationURL    string
	integrationAPIKey string
)

func TestMain(main *testing.M) {
	integrationURL = strings.TrimRight(os.Getenv("GYRIFI_TEST_QDRANT_URL"), "/")
	integrationAPIKey = os.Getenv("GYRIFI_TEST_QDRANT_API_KEY")
	if integrationURL != "" {
		if err := waitForIntegrationQdrant(); err != nil {
			log.Printf("Qdrant integration setup failed: %v", err)
			os.Exit(1)
		}
		version, err := qdrantVersion()
		if err != nil {
			log.Printf("Qdrant version query failed: %v", err)
			os.Exit(1)
		}
		log.Printf("Qdrant integration version: %s", version)
		sweepStaleCollections()
	}
	os.Exit(main.Run())
}

func requireIntegration(t *testing.T) {
	t.Helper()
	if integrationURL == "" {
		t.Skip("GYRIFI_TEST_QDRANT_URL is not set")
	}
}

func integrationRequest(ctx context.Context, method, path string, body []byte, apiKey string) ([]byte, int, error) {
	request, err := http.NewRequestWithContext(ctx, method, integrationURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	request.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		request.Header.Set("api-key", apiKey)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	value, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	return value, response.StatusCode, err
}

func waitForIntegrationQdrant() error {
	deadline := time.Now().Add(30 * time.Second)
	for {
		_, status, err := integrationRequest(context.Background(), http.MethodGet, "/", nil, integrationAPIKey)
		if err == nil && status >= 200 && status < 300 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("not ready after 30s: status %d: %w", status, err)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func qdrantVersion() (string, error) {
	body, status, err := integrationRequest(context.Background(), http.MethodGet, "/", nil, integrationAPIKey)
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("status %d", status)
	}
	var response struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}
	if response.Version == "" {
		return "", errors.New("Qdrant response omitted version")
	}
	return response.Version, nil
}

func uniqueCollection(t *testing.T) string {
	t.Helper()
	value := make([]byte, 8)
	if _, err := rand.Read(value); err != nil {
		t.Fatal(err)
	}
	return "gyrifi_it_" + hex.EncodeToString(value)
}

func newIntegrationAdapter(t *testing.T, distance string, size int) *Adapter {
	t.Helper()
	requireIntegration(t)
	collection := uniqueCollection(t)
	body, _ := json.Marshal(map[string]any{"vectors": map[string]any{"size": size, "distance": distance}})
	_, status, err := integrationRequest(context.Background(), http.MethodPut, "/collections/"+url.PathEscape(collection), body, integrationAPIKey)
	if err != nil || status < 200 || status >= 300 {
		t.Fatalf("create collection %s: status %d: %v", collection, status, err)
	}
	t.Cleanup(func() {
		_, cleanupStatus, cleanupErr := integrationRequest(context.Background(), http.MethodDelete, "/collections/"+url.PathEscape(collection), nil, integrationAPIKey)
		if cleanupErr != nil || (cleanupStatus != http.StatusOK && cleanupStatus != http.StatusNotFound) {
			t.Errorf("drop collection %s: status %d: %v", collection, cleanupStatus, cleanupErr)
		}
	})
	adapter, err := New(integrationURL, collection, integrationAPIKey)
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}

func sweepStaleCollections() {
	body, status, err := integrationRequest(context.Background(), http.MethodGet, "/collections", nil, integrationAPIKey)
	if err != nil || status < 200 || status >= 300 {
		log.Printf("Qdrant stale-collection sweep skipped: status %d: %v", status, err)
		return
	}
	var response struct {
		Result struct {
			Collections []struct {
				Name string `json:"name"`
			} `json:"collections"`
		} `json:"result"`
	}
	if json.Unmarshal(body, &response) != nil {
		return
	}
	for _, collection := range response.Result.Collections {
		if strings.HasPrefix(collection.Name, "gyrifi_it_") {
			_, _, _ = integrationRequest(context.Background(), http.MethodDelete, "/collections/"+url.PathEscape(collection.Name), nil, integrationAPIKey)
		}
	}
}

func compilePut(t *testing.T, adapter *Adapter, unit string, desired json.RawMessage) targets.Plan {
	t.Helper()
	plan, err := adapter.Compile(context.Background(), []ledger.Change{{Unit: unit, Action: ledger.ChangePut, Desired: desired, DesiredFingerprint: ledger.Fingerprint(desired)}})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestIntegrationRoundTripMetricsAndPayloadTypes(t *testing.T) {
	for _, distance := range []string{"Cosine", "Dot", "Euclid"} {
		distance := distance
		t.Run(distance, func(t *testing.T) {
			t.Parallel()
			adapter := newIntegrationAdapter(t, distance, 3)
			desired := json.RawMessage(`{"payload":{"z":[1,"two",false,null,{"nested":true}],"number":3.5,"string":"value","bool":true,"null":null,"object":{"b":2,"a":1}},"vector":[3,4,0],"id":1}`)
			plan := compilePut(t, adapter, "1", desired)
			if err := adapter.Apply(context.Background(), plan); err != nil {
				t.Fatal(err)
			}
			value, err := adapter.Read(context.Background(), "1")
			if err != nil {
				t.Fatal(err)
			}
			fingerprint, err := adapter.Fingerprint(context.Background(), "1")
			if err != nil {
				t.Fatal(err)
			}
			if !value.Exists || fingerprint != value.Fingerprint || !equivalentJSON(desired, value.Value, distance) {
				t.Fatalf("round trip value = %#v, fingerprint = %q", value, fingerprint)
			}
			if err := adapter.Verify(context.Background(), plan); err != nil {
				t.Fatalf("verify %s round trip: %v", distance, err)
			}
			plan.Operations[0].ExpectedExists = true
			plan.Operations[0].ExpectedFingerprint = fingerprint
			if err := adapter.Apply(context.Background(), plan); err != nil {
				t.Fatalf("reapply identical point: %v", err)
			}
			after, err := adapter.Fingerprint(context.Background(), "1")
			if err != nil || after != fingerprint {
				t.Fatalf("fingerprint after reapply = %q, %v; want %q", after, err, fingerprint)
			}
		})
	}
}

func TestIntegrationDeleteExistingAndAbsent(t *testing.T) {
	adapter := newIntegrationAdapter(t, "Cosine", 3)
	put := compilePut(t, adapter, "1", json.RawMessage(`{"id":1,"vector":[1,0,0],"payload":{"value":"present"}}`))
	if err := adapter.Apply(context.Background(), put); err != nil {
		t.Fatal(err)
	}
	current, err := adapter.Read(context.Background(), "1")
	if err != nil {
		t.Fatal(err)
	}
	remove, err := adapter.Compile(context.Background(), []ledger.Change{{Unit: "1", Action: ledger.ChangeDelete}})
	if err != nil {
		t.Fatal(err)
	}
	remove.Operations[0].ExpectedExists = true
	remove.Operations[0].ExpectedFingerprint = current.Fingerprint
	if err := adapter.Apply(context.Background(), remove); err != nil {
		t.Fatal(err)
	}
	removed, err := adapter.Read(context.Background(), "1")
	if err != nil || removed.Exists {
		t.Fatalf("removed point = %#v, %v", removed, err)
	}
	absent, err := adapter.Compile(context.Background(), []ledger.Change{{Unit: "2", Action: ledger.ChangeDelete}})
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.Apply(context.Background(), absent); err != nil {
		t.Fatalf("delete absent point: %v", err)
	}
}

func TestIntegrationFailureClassifications(t *testing.T) {
	requireIntegration(t)
	t.Run("missing collection", func(t *testing.T) {
		collection := uniqueCollection(t)
		t.Cleanup(func() {
			_, _, _ = integrationRequest(context.Background(), http.MethodDelete, "/collections/"+collection, nil, integrationAPIKey)
		})
		adapter, err := New(integrationURL, collection, integrationAPIKey)
		if err != nil {
			t.Fatal(err)
		}
		_, err = adapter.Compile(context.Background(), []ledger.Change{{Unit: "1", Action: ledger.ChangeDelete}})
		if !errors.Is(err, targets.ErrNotFound) {
			t.Fatalf("missing collection error = %v", err)
		}
	})

	t.Run("wrong dimension", func(t *testing.T) {
		adapter := newIntegrationAdapter(t, "Cosine", 3)
		plan := compilePut(t, adapter, "1", json.RawMessage(`{"id":1,"vector":[1,2],"payload":{}}`))
		if err := adapter.Apply(context.Background(), plan); !errors.Is(err, targets.ErrSemantic) {
			t.Fatalf("dimension error = %v", err)
		}
	})

	t.Run("unreachable", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		address := listener.Addr().String()
		_ = listener.Close()
		adapter, err := New("http://"+address, "missing", "")
		if err != nil {
			t.Fatal(err)
		}
		if err := adapter.Health(context.Background()); !errors.Is(err, targets.ErrUnavailable) {
			t.Fatalf("unreachable error = %v", err)
		}
	})

	t.Run("authentication", func(t *testing.T) {
		if integrationAPIKey == "" {
			t.Skip("GYRIFI_TEST_QDRANT_API_KEY is not set")
		}
		adapter := newIntegrationAdapter(t, "Cosine", 3)
		wrongKey := integrationAPIKey + "-wrong"
		unauthorized, err := New(integrationURL, adapter.collection, wrongKey)
		if err != nil {
			t.Fatal(err)
		}
		err = unauthorized.Health(context.Background())
		if !errors.Is(err, targets.ErrAuthentication) || strings.Contains(err.Error(), integrationAPIKey) || strings.Contains(err.Error(), wrongKey) {
			t.Fatalf("authentication error = %v", err)
		}
	})
}

func TestIntegrationDriftAndPartialPlanFailure(t *testing.T) {
	t.Run("drift", func(t *testing.T) {
		t.Parallel()
		adapter := newIntegrationAdapter(t, "Cosine", 3)
		desired := json.RawMessage(`{"id":1,"vector":[1,0,0],"payload":{"state":"expected"}}`)
		plan := compilePut(t, adapter, "1", desired)
		if err := adapter.Apply(context.Background(), plan); err != nil {
			t.Fatal(err)
		}
		body := []byte(`{"points":[{"id":1,"vector":[1,0,0],"payload":{"state":"drifted"}}]}`)
		_, status, err := integrationRequest(context.Background(), http.MethodPut, "/collections/"+url.PathEscape(adapter.collection)+"/points?wait=true", body, integrationAPIKey)
		if err != nil || status < 200 || status >= 300 {
			t.Fatalf("mutate point: status %d: %v", status, err)
		}
		err = adapter.Verify(context.Background(), plan)
		var verification *targets.VerificationError
		if !errors.As(err, &verification) || len(verification.Mismatches) != 1 || verification.Mismatches[0].Unit != "1" {
			t.Fatalf("drift verification = %#v", err)
		}
	})

	t.Run("partial plan", func(t *testing.T) {
		t.Parallel()
		adapter := newIntegrationAdapter(t, "Cosine", 3)
		plan, err := adapter.Compile(context.Background(), []ledger.Change{
			{Unit: "1", Action: ledger.ChangePut, Desired: json.RawMessage(`{"id":1,"vector":[1,0,0],"payload":{"valid":true}}`)},
			{Unit: "2", Action: ledger.ChangePut, Desired: json.RawMessage(`{"id":2,"vector":[1,0],"payload":{"valid":false}}`)},
		})
		if err != nil {
			t.Fatal(err)
		}
		err = adapter.Apply(context.Background(), plan)
		if !errors.Is(err, targets.ErrSemantic) {
			t.Fatalf("partial-plan error = %v", err)
		}
		first, firstErr := adapter.Read(context.Background(), "1")
		second, secondErr := adapter.Read(context.Background(), "2")
		if firstErr != nil || secondErr != nil || !first.Exists || second.Exists {
			t.Fatalf("partial result: first=%#v (%v), second=%#v (%v)", first, firstErr, second, secondErr)
		}
	})
}

func TestIntegrationEngineReleaseAndRollback(t *testing.T) {
	adapter := newIntegrationAdapter(t, "Cosine", 3)
	ctx := context.Background()
	before := json.RawMessage(`{"id":42,"vector":[1,0,0],"payload":{"version":"before"}}`)
	directory := t.TempDir()
	repo, err := repository.OpenSQLite(ctx, filepath.Join(directory, "state.db"), filepath.Join(directory, "objects"))
	if err != nil {
		t.Fatal(err)
	}
	application := engine.New(repo, adapter, nil)
	t.Cleanup(func() { _ = application.Close() })
	ledgerValue, err := application.CreateLedger(ctx, "Qdrant integration", "")
	if err != nil {
		t.Fatal(err)
	}
	releaseValue := func(title, idempotencyKey string, value json.RawMessage) ledger.Release {
		change, err := application.CreateChange(ctx, ledgerValue.ID, engine.CreateChangeRequest{Unit: "42", Action: ledger.ChangePut, Desired: value, IdempotencyKey: idempotencyKey})
		if err != nil {
			t.Fatal(err)
		}
		proposal, err := application.CreateProposal(ctx, ledgerValue.ID, engine.CreateProposalRequest{Title: title, ChangeIDs: []string{change.ID}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := application.EvaluateProposal(ctx, ledgerValue.ID, proposal.ID, "round trip"); err != nil {
			t.Fatal(err)
		}
		if err := application.ApproveProposal(ctx, ledgerValue.ID, proposal.ID, "integration-test"); err != nil {
			t.Fatal(err)
		}
		release, err := application.ReleaseProposal(ctx, ledgerValue.ID, proposal.ID)
		if err != nil {
			t.Fatal(err)
		}
		return release
	}
	baselineRelease := releaseValue("Baseline Qdrant release", "qdrant-integration-baseline", before)
	desired := json.RawMessage(`{"id":42,"vector":[0,3,4],"payload":{"version":"after"}}`)
	_ = releaseValue("Real Qdrant release", "qdrant-integration-update", desired)
	afterRelease, err := adapter.Read(ctx, "42")
	if err != nil || !equivalentJSON(desired, afterRelease.Value, "Cosine") {
		t.Fatalf("released value = %#v, %v", afterRelease, err)
	}

	intents, err := repo.ListReleaseIntentsForLedger(ctx, ledgerValue.ID, nil)
	if err != nil || len(intents) != 2 {
		t.Fatalf("release intents = %#v, %v", intents, err)
	}
	var plan targets.Plan
	if err := json.Unmarshal(intents[0].Plan, &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Operations) != 1 || !plan.Operations[0].BeforeExists || plan.Operations[0].BeforeObjectHash == "" {
		t.Fatalf("before-image operation = %#v", plan.Operations)
	}
	retained, err := repo.ReadObject(ctx, plan.Operations[0].BeforeObjectHash)
	if err != nil || !equivalentJSON(before, retained, "Cosine") {
		t.Fatalf("retained before-image = %s, %v", retained, err)
	}

	rollback, err := application.CreateRollbackProposal(ctx, ledgerValue.ID, baselineRelease.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.EvaluateProposal(ctx, ledgerValue.ID, rollback.ID, "restore prior value"); err != nil {
		t.Fatal(err)
	}
	if err := application.ApproveProposal(ctx, ledgerValue.ID, rollback.ID, "integration-test"); err != nil {
		t.Fatal(err)
	}
	if _, err := application.ReleaseProposal(ctx, ledgerValue.ID, rollback.ID); err != nil {
		t.Fatal(err)
	}
	afterRollback, err := adapter.Read(ctx, "42")
	if err != nil || !equivalentJSON(before, afterRollback.Value, "Cosine") {
		t.Fatalf("rolled-back value = %#v, %v", afterRollback, err)
	}
}
