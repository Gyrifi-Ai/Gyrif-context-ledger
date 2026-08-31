package httpinterface

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
)

//go:embed static
var studio embed.FS

type Server struct {
	engine   *engine.Engine
	logger   *slog.Logger
	version  string
	requests atomic.Uint64
	handler  http.Handler
}
type apiError struct {
	Error struct {
		Code    engine.ErrorCode `json:"code"`
		Message string           `json:"message"`
	} `json:"error"`
}

func New(application *engine.Engine, logger *slog.Logger, version string) *Server {
	server := &Server{engine: application, logger: logger, version: version}
	mux := http.NewServeMux()
	server.routes(mux)
	server.handler = server.middleware(mux)
	return server
}
func (server *Server) Handler() http.Handler { return server.handler }
func (server *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/system/status", server.status)
	mux.HandleFunc("GET /api/v1/adapters", server.adapters)
	mux.HandleFunc("GET /api/v1/ledgers", server.listLedgers)
	mux.HandleFunc("POST /api/v1/ledgers", server.createLedger)
	mux.HandleFunc("GET /api/v1/ledgers/{ledgerID}/changes", server.listChanges)
	mux.HandleFunc("POST /api/v1/ledgers/{ledgerID}/changes", server.createChange)
	mux.HandleFunc("GET /api/v1/ledgers/{ledgerID}/proposals", server.listProposals)
	mux.HandleFunc("POST /api/v1/ledgers/{ledgerID}/proposals", server.createProposal)
	mux.HandleFunc("GET /api/v1/ledgers/{ledgerID}/proposals/{proposalID}", server.proposalDetail)
	mux.HandleFunc("GET /api/v1/ledgers/{ledgerID}/proposals/{proposalID}/checks", server.proposalChecks)
	mux.HandleFunc("GET /api/v1/ledgers/{ledgerID}/proposals/{proposalID}/approvals", server.proposalApprovals)
	mux.HandleFunc("POST /api/v1/ledgers/{ledgerID}/proposals/{proposalID}/evaluation", server.evaluateProposal)
	mux.HandleFunc("POST /api/v1/ledgers/{ledgerID}/proposals/{proposalID}/approvals", server.approveProposal)
	mux.HandleFunc("POST /api/v1/ledgers/{ledgerID}/proposals/{proposalID}/release", server.releaseProposal)
	mux.HandleFunc("GET /api/v1/ledgers/{ledgerID}/release-intents", server.listReleaseIntents)
	mux.HandleFunc("GET /api/v1/ledgers/{ledgerID}/release-intents/{intentID}", server.releaseIntentDetail)
	mux.HandleFunc("POST /api/v1/ledgers/{ledgerID}/release-intents/{intentID}/retry", server.retryReleaseIntent)
	mux.HandleFunc("POST /api/v1/ledgers/{ledgerID}/release-intents/{intentID}/resolve", server.resolveReleaseIntent)
	mux.HandleFunc("GET /api/v1/ledgers/{ledgerID}/releases", server.listReleases)
	mux.HandleFunc("POST /api/v1/ledgers/{ledgerID}/releases/{releaseID}/rollback", server.rollbackRelease)
	mux.HandleFunc("GET /events/v1", server.events)
	mux.Handle("/", server.studioHandler())
}
func (server *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestID := fmt.Sprintf("req-%d", server.requests.Add(1))
		writer.Header().Set("X-Request-ID", requestID)
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		start := time.Now()
		next.ServeHTTP(writer, request)
		server.logger.InfoContext(request.Context(), "http request", "request_id", requestID, "method", request.Method, "path", request.URL.Path, "duration", time.Since(start))
	})
}
func (server *Server) decode(writer http.ResponseWriter, request *http.Request, value any) bool {
	request.Body = http.MaxBytesReader(writer, request.Body, 4<<20)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		server.writeError(writer, engine.CodeInvalid, "Request body is invalid.", http.StatusBadRequest)
		return false
	}
	return true
}
func (server *Server) writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
func (server *Server) writeEngineError(writer http.ResponseWriter, err error) {
	code, message := engine.PublicError(err)
	status := http.StatusInternalServerError
	switch code {
	case engine.CodeInvalid:
		status = http.StatusBadRequest
	case engine.CodeNotFound:
		status = http.StatusNotFound
	case engine.CodeConflict:
		status = http.StatusConflict
	case engine.CodeUnavailable:
		status = http.StatusServiceUnavailable
	}
	server.logger.Error("request failed", "code", code, "error", err)
	server.writeError(writer, code, message, status)
}
func (server *Server) writeError(writer http.ResponseWriter, code engine.ErrorCode, message string, status int) {
	var body apiError
	body.Error.Code = code
	body.Error.Message = message
	server.writeJSON(writer, status, body)
}
func (server *Server) status(writer http.ResponseWriter, _ *http.Request) {
	server.writeJSON(writer, http.StatusOK, map[string]any{"status": "ok", "version": server.version, "inference": server.engine.InferenceName()})
}
func (server *Server) adapters(writer http.ResponseWriter, _ *http.Request) {
	server.writeJSON(writer, http.StatusOK, map[string]any{"items": []any{map[string]any{"id": "qdrant", "name": "Qdrant", "capabilities": server.engine.TargetCapabilities()}}})
}
func (server *Server) listLedgers(writer http.ResponseWriter, request *http.Request) {
	items, err := server.engine.ListLedgers(request.Context())
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusOK, map[string]any{"items": items})
}
func (server *Server) createLedger(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if !server.decode(writer, request, &input) {
		return
	}
	value, err := server.engine.CreateLedger(request.Context(), input.Name, input.Description)
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusCreated, value)
}
func (server *Server) listChanges(writer http.ResponseWriter, request *http.Request) {
	items, err := server.engine.ListChanges(request.Context(), request.PathValue("ledgerID"))
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusOK, map[string]any{"items": items})
}
func (server *Server) createChange(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Unit           string              `json:"unit"`
		Action         ledger.ChangeAction `json:"action"`
		Desired        json.RawMessage     `json:"desired"`
		IdempotencyKey string              `json:"idempotencyKey"`
	}
	if !server.decode(writer, request, &input) {
		return
	}
	value, err := server.engine.CreateChange(request.Context(), request.PathValue("ledgerID"), engine.CreateChangeRequest{Unit: input.Unit, Action: input.Action, Desired: input.Desired, IdempotencyKey: input.IdempotencyKey})
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusAccepted, value)
}
func (server *Server) listProposals(writer http.ResponseWriter, request *http.Request) {
	items, err := server.engine.ListProposals(request.Context(), request.PathValue("ledgerID"))
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusOK, map[string]any{"items": items})
}
func (server *Server) createProposal(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Title     string   `json:"title"`
		ChangeIDs []string `json:"changeIds"`
	}
	if !server.decode(writer, request, &input) {
		return
	}
	value, err := server.engine.CreateProposal(request.Context(), request.PathValue("ledgerID"), engine.CreateProposalRequest{Title: input.Title, ChangeIDs: input.ChangeIDs})
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusCreated, value)
}
func (server *Server) proposalDetail(writer http.ResponseWriter, request *http.Request) {
	value, err := server.engine.LoadProposalDetail(request.Context(), request.PathValue("ledgerID"), request.PathValue("proposalID"))
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusOK, value)
}
func (server *Server) proposalChecks(writer http.ResponseWriter, request *http.Request) {
	// TODO(GRF-220): protect evidence reads with the same authorisation as Change reads.
	items, err := server.engine.ListCheckResults(request.Context(), request.PathValue("ledgerID"), request.PathValue("proposalID"))
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusOK, map[string]any{"items": items})
}
func (server *Server) proposalApprovals(writer http.ResponseWriter, request *http.Request) {
	items, err := server.engine.ListApprovals(request.Context(), request.PathValue("ledgerID"), request.PathValue("proposalID"))
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusOK, map[string]any{"items": items})
}
func (server *Server) evaluateProposal(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Criteria string `json:"criteria"`
	}
	if !server.decode(writer, request, &input) {
		return
	}
	value, err := server.engine.EvaluateProposal(request.Context(), request.PathValue("ledgerID"), request.PathValue("proposalID"), input.Criteria)
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusOK, value)
}
func (server *Server) approveProposal(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Actor string `json:"actor"`
	}
	if !server.decode(writer, request, &input) {
		return
	}
	if err := server.engine.ApproveProposal(request.Context(), request.PathValue("ledgerID"), request.PathValue("proposalID"), input.Actor); err != nil {
		server.writeEngineError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
func (server *Server) releaseProposal(writer http.ResponseWriter, request *http.Request) {
	value, err := server.engine.ReleaseProposal(request.Context(), request.PathValue("ledgerID"), request.PathValue("proposalID"))
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusCreated, value)
}
func (server *Server) listReleaseIntents(writer http.ResponseWriter, request *http.Request) {
	var status *ledger.ReleaseIntentStatus
	if rawStatus := request.URL.Query().Get("status"); rawStatus != "" {
		value := ledger.ReleaseIntentStatus(rawStatus)
		if !engine.ValidReleaseIntentStatus(value) {
			server.writeError(writer, engine.CodeInvalid, "Release Intent status is invalid.", http.StatusBadRequest)
			return
		}
		status = &value
	}
	items, err := server.engine.ListReleaseIntents(request.Context(), request.PathValue("ledgerID"), status)
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusOK, map[string]any{"items": items})
}
func (server *Server) releaseIntentDetail(writer http.ResponseWriter, request *http.Request) {
	value, err := server.engine.LoadReleaseIntent(request.Context(), request.PathValue("ledgerID"), request.PathValue("intentID"))
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusOK, value)
}
func (server *Server) retryReleaseIntent(writer http.ResponseWriter, request *http.Request) {
	value, err := server.engine.RetryReleaseIntent(request.Context(), request.PathValue("ledgerID"), request.PathValue("intentID"))
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusOK, value)
}
func (server *Server) resolveReleaseIntent(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Resolution string `json:"resolution"`
		Note       string `json:"note"`
	}
	if !server.decode(writer, request, &input) {
		return
	}
	if err := server.engine.ResolveReleaseIntent(request.Context(), request.PathValue("ledgerID"), request.PathValue("intentID"), input.Resolution, input.Note); err != nil {
		server.writeEngineError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
func (server *Server) listReleases(writer http.ResponseWriter, request *http.Request) {
	items, err := server.engine.ListReleases(request.Context(), request.PathValue("ledgerID"))
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusOK, map[string]any{"items": items})
}
func (server *Server) rollbackRelease(writer http.ResponseWriter, request *http.Request) {
	value, err := server.engine.CreateRollbackProposal(request.Context(), request.PathValue("ledgerID"), request.PathValue("releaseID"))
	if err != nil {
		server.writeEngineError(writer, err)
		return
	}
	server.writeJSON(writer, http.StatusCreated, value)
}
func (server *Server) events(writer http.ResponseWriter, request *http.Request) {
	flusher, ok := writer.(http.Flusher)
	if !ok {
		server.writeError(writer, engine.CodeInternal, "Events are unavailable.", http.StatusInternalServerError)
		return
	}
	events, unsubscribe := server.engine.Events().Subscribe(16)
	defer unsubscribe()
	ledgerID := request.URL.Query().Get("ledgerId")
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	if _, err := fmt.Fprint(writer, "event: ledger\ndata: {\"status\":\"connected\"}\n\n"); err != nil {
		return
	}
	flusher.Flush()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-request.Context().Done():
			return
		case event, open := <-events:
			if !open {
				return
			}
			if ledgerID != "" && event.LedgerID != ledgerID {
				continue
			}
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", event.Kind, data); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			if _, err := fmt.Fprint(writer, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
func (server *Server) studioHandler() http.Handler {
	contents, err := fs.Sub(studio, "static")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(contents))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api/") || strings.HasPrefix(request.URL.Path, "/events/") {
			server.writeError(writer, engine.CodeNotFound, "Endpoint was not found.", http.StatusNotFound)
			return
		}
		requested := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
		if requested == "." || requested == "" {
			requested = "index.html"
		}
		if _, err := fs.Stat(contents, requested); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				server.writeError(writer, engine.CodeInternal, "Studio is unavailable.", http.StatusInternalServerError)
				return
			}
			request.URL.Path = "/index.html"
		}
		files.ServeHTTP(writer, request)
	})
}

var _ = context.Canceled
