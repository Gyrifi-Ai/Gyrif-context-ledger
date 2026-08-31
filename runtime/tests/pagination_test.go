package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
	httpinterface "github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
)

type changePageResponse struct {
	Items      []ledger.Change `json:"items"`
	NextCursor string          `json:"nextCursor"`
}

func listRequest(t *testing.T, application *engine.Engine, path string, result any) *httptest.ResponseRecorder {
	t.Helper()
	server := httpinterface.New(application, slog.New(slog.NewTextHandler(io.Discard, nil)))
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if result != nil && response.Code == http.StatusOK {
		if err := json.Unmarshal(response.Body.Bytes(), result); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	return response
}

func insertPageChange(t *testing.T, repo *repository.SQLite, ledgerID, id string, createdAt time.Time, status ledger.ChangeStatus, action ledger.ChangeAction) {
	t.Helper()
	change := ledger.Change{
		ID:                 id,
		LedgerID:           ledgerID,
		Unit:               "unit-" + id,
		Action:             action,
		Desired:            json.RawMessage(`{"value":true}`),
		DesiredFingerprint: ledger.Fingerprint([]byte(`{"value":true}`)),
		IdempotencyKey:     "request-" + id,
		RequestFingerprint: "sha256:request-" + id,
		Status:             status,
		CreatedAt:          createdAt,
	}
	if action == ledger.ChangeDelete {
		change.Desired = nil
		change.DesiredFingerprint = ledger.Fingerprint(nil)
	}
	if err := repo.InsertChange(context.Background(), &change); err != nil {
		t.Fatal(err)
	}
}

func TestChangePaginationTraversesStableSnapshot(t *testing.T) {
	application, repo, ledgerID := newProposalDetailEngine(t)
	base := time.Date(2026, time.August, 31, 12, 0, 0, 123, time.UTC)
	expected := make(map[string]bool, 205)
	for index := 0; index < 205; index++ {
		id := fmt.Sprintf("chg_page_%03d", index)
		expected[id] = true
		insertPageChange(t, repo, ledgerID, id, base.Add(time.Duration(index)*time.Second), ledger.ChangeReady, ledger.ChangePut)
	}

	cursor := ""
	seen := make(map[string]bool, 205)
	pageSizes := make([]int, 0, 5)
	var previous *ledger.Change
	for {
		path := "/api/v1/ledgers/" + ledgerID + "/changes?limit=50"
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		var page changePageResponse
		response := listRequest(t, application, path, &page)
		if response.Code != http.StatusOK {
			t.Fatalf("page response = %d %s", response.Code, response.Body.String())
		}
		pageSizes = append(pageSizes, len(page.Items))
		for index := range page.Items {
			item := page.Items[index]
			if seen[item.ID] {
				t.Fatalf("duplicate Change %s", item.ID)
			}
			if !expected[item.ID] {
				t.Fatalf("post-cursor Change appeared: %s", item.ID)
			}
			if previous != nil && (item.CreatedAt.After(previous.CreatedAt) || (item.CreatedAt.Equal(previous.CreatedAt) && item.ID >= previous.ID)) {
				t.Fatalf("unstable order: %s before %s", previous.ID, item.ID)
			}
			seen[item.ID] = true
			copy := item
			previous = &copy
		}
		if len(pageSizes) == 1 {
			for index := 0; index < 3; index++ {
				insertPageChange(t, repo, ledgerID, fmt.Sprintf("chg_new_%d", index), base.Add(time.Hour+time.Duration(index)*time.Second), ledger.ChangeReady, ledger.ChangePut)
			}
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if fmt.Sprint(pageSizes) != "[50 50 50 50 5]" {
		t.Fatalf("page sizes = %v", pageSizes)
	}
	if len(seen) != len(expected) {
		t.Fatalf("visited %d of %d pre-existing Changes", len(seen), len(expected))
	}
}

func TestChangePaginationUsesIDTiebreakerAndSQLFilters(t *testing.T) {
	application, repo, ledgerID := newProposalDetailEngine(t)
	createdAt := time.Date(2026, time.August, 31, 13, 0, 0, 456, time.UTC)
	for _, id := range []string{"chg_tie_a", "chg_tie_c", "chg_tie_b"} {
		insertPageChange(t, repo, ledgerID, id, createdAt, ledger.ChangeReady, ledger.ChangePut)
	}
	var first changePageResponse
	listRequest(t, application, "/api/v1/ledgers/"+ledgerID+"/changes?limit=2", &first)
	var second changePageResponse
	listRequest(t, application, "/api/v1/ledgers/"+ledgerID+"/changes?limit=2&cursor="+url.QueryEscape(first.NextCursor), &second)
	ids := []string{first.Items[0].ID, first.Items[1].ID, second.Items[0].ID}
	if !sort.StringsAreSorted([]string{ids[2], ids[1], ids[0]}) || strings.Join(ids, ",") != "chg_tie_c,chg_tie_b,chg_tie_a" {
		t.Fatalf("tie order = %v", ids)
	}

	insertPageChange(t, repo, ledgerID, "chg_filter_delete", createdAt.Add(time.Second), ledger.ChangeReady, ledger.ChangeDelete)
	insertPageChange(t, repo, ledgerID, "chg_filter_released", createdAt.Add(2*time.Second), ledger.ChangeReleased, ledger.ChangeDelete)
	var filtered changePageResponse
	listRequest(t, application, "/api/v1/ledgers/"+ledgerID+"/changes?status=READY&action=DELETE", &filtered)
	if len(filtered.Items) != 1 || filtered.Items[0].ID != "chg_filter_delete" {
		t.Fatalf("filtered Changes = %#v", filtered.Items)
	}
}

func TestListValidationAndLedgerPaging(t *testing.T) {
	application, _, ledgerID := newProposalDetailEngine(t)
	cases := []struct {
		path    string
		message string
	}{
		{"/api/v1/ledgers?limit=0", "Limit must be between 1 and 200."},
		{"/api/v1/ledgers?limit=201", "Limit must be between 1 and 200."},
		{"/api/v1/ledgers?limit=nope", "Limit must be between 1 and 200."},
		{"/api/v1/ledgers?cursor=", "The cursor is not valid."},
		{"/api/v1/ledgers/" + ledgerID + "/changes?cursor=garbage", "The cursor is not valid."},
		{"/api/v1/ledgers/" + ledgerID + "/changes?status=UNKNOWN", "Status must be one of: ACCEPTED, READY, INVALID, RELEASED."},
		{"/api/v1/ledgers/" + ledgerID + "/changes?action=PATCH", "Action must be one of: PUT, DELETE."},
		{"/api/v1/ledgers/" + ledgerID + "/proposals?status=UNKNOWN", "Status must be one of: DRAFT, REVIEWED, APPROVED, RELEASED, BLOCKED, CANCELLED."},
	}
	for _, test := range cases {
		response := listRequest(t, application, test.path, nil)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"INVALID_ARGUMENT"`) || !strings.Contains(response.Body.String(), test.message) {
			t.Errorf("%s = %d %s", test.path, response.Code, response.Body.String())
		}
	}

	for index := 0; index < 50; index++ {
		if _, err := application.CreateLedger(context.Background(), fmt.Sprintf("Page ledger %02d", index), ""); err != nil {
			t.Fatal(err)
		}
	}
	var first struct {
		Items      []ledger.Ledger `json:"items"`
		NextCursor string          `json:"nextCursor"`
	}
	listRequest(t, application, "/api/v1/ledgers", &first)
	if len(first.Items) != 50 || first.NextCursor == "" {
		t.Fatalf("first Ledger page = %d, cursor %q", len(first.Items), first.NextCursor)
	}
	var second struct {
		Items      []ledger.Ledger `json:"items"`
		NextCursor string          `json:"nextCursor"`
	}
	listRequest(t, application, "/api/v1/ledgers?cursor="+url.QueryEscape(first.NextCursor), &second)
	if len(second.Items) != 1 || second.NextCursor != "" {
		t.Fatalf("second Ledger page = %d, cursor %q", len(second.Items), second.NextCursor)
	}
}
