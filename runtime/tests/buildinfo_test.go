package tests

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/buildinfo"
	httpinterface "github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http"
)

func TestSystemStatusUsesBuildInfo(t *testing.T) {
	originalVersion, originalCommit, originalDate := buildinfo.Version, buildinfo.Commit, buildinfo.Date
	t.Cleanup(func() {
		buildinfo.Version, buildinfo.Commit, buildinfo.Date = originalVersion, originalCommit, originalDate
	})
	buildinfo.Version = "9.9.9"
	buildinfo.Commit = "deadbeef"
	buildinfo.Date = "2026-09-01T00:00:00Z"

	application, _, _ := newProposalDetailEngine(t)
	server := httpinterface.New(application, slog.New(slog.NewTextHandler(io.Discard, nil)))
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body struct {
		Status    string `json:"status"`
		Version   string `json:"version"`
		Commit    string `json:"commit"`
		BuildDate string `json:"buildDate"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" || body.Version != buildinfo.Version || body.Commit != buildinfo.Commit || body.BuildDate != buildinfo.Date {
		t.Fatalf("status body = %#v", body)
	}
}
