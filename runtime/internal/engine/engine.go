package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets"
)

type ErrorCode string

const (
	CodeInvalid     ErrorCode = "INVALID_ARGUMENT"
	CodeNotFound    ErrorCode = "NOT_FOUND"
	CodeConflict    ErrorCode = "CONFLICT"
	CodeUnavailable ErrorCode = "UNAVAILABLE"
	CodeInternal    ErrorCode = "INTERNAL"
)

type Error struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (err *Error) Error() string { return err.Message }
func (err *Error) Unwrap() error { return err.Err }
func wrap(code ErrorCode, message string, err error) error {
	return &Error{Code: code, Message: message, Err: err}
}
func PublicError(err error) (ErrorCode, string) {
	var engineError *Error
	if errors.As(err, &engineError) {
		return engineError.Code, engineError.Message
	}
	return CodeInternal, "The operation could not be completed."
}

type Engine struct {
	repository   repository.Repository
	target       targets.TargetAdapter
	targetHealth any
	inference    inference.Provider
	metrics      MetricSink
	releaseMu    sync.Mutex
	events       *Broker
}

func New(repo repository.Repository, target targets.TargetAdapter, provider inference.Provider, sinks ...MetricSink) *Engine {
	var metrics MetricSink = discardMetrics{}
	if len(sinks) > 0 && sinks[0] != nil {
		metrics = sinks[0]
	}
	metered := target
	if target != nil {
		metered = &meteredTarget{target: target, metrics: metrics}
	}
	return &Engine{repository: repo, target: metered, targetHealth: target, inference: provider, metrics: metrics, events: &Broker{}}
}
func (engine *Engine) InferenceName() string {
	if engine.inference == nil {
		return "disabled"
	}
	return engine.inference.Name()
}
func (engine *Engine) Close() error { return engine.repository.Close() }

func (engine *Engine) CreateLedger(ctx context.Context, name, description string) (ledger.Ledger, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ledger.Ledger{}, wrap(CodeInvalid, "Ledger name is required.", ledger.ErrInvalid)
	}
	id, err := ledger.NewID("ldg")
	if err != nil {
		return ledger.Ledger{}, wrap(CodeInternal, "Could not create the ledger.", err)
	}
	value := ledger.Ledger{ID: id, Name: name, Description: strings.TrimSpace(description), CreatedAt: time.Now().UTC()}
	if err := engine.repository.CreateLedger(ctx, value); err != nil {
		return ledger.Ledger{}, wrap(CodeConflict, "A ledger with that name already exists.", err)
	}
	return value, nil
}
func (engine *Engine) TargetCapabilities() targets.Capabilities { return engine.target.Capabilities() }
func ensureLedgerID(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("ledger id is required")
	}
	return nil
}
