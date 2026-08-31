package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
)

type CreateChangeRequest struct {
	Unit           string
	Action         ledger.ChangeAction
	Desired        json.RawMessage
	IdempotencyKey string
}

func (engine *Engine) CreateChange(ctx context.Context, ledgerID string, request CreateChangeRequest) (ledger.Change, error) {
	if err := ensureLedgerID(ledgerID); err != nil {
		return ledger.Change{}, wrap(CodeInvalid, "Ledger is required.", err)
	}
	request.Unit = strings.TrimSpace(request.Unit)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if request.IdempotencyKey == "" {
		return ledger.Change{}, wrap(CodeInvalid, "Idempotency key is required.", ledger.ErrInvalid)
	}
	if request.Action == ledger.ChangePut {
		var compact bytes.Buffer
		if err := json.Compact(&compact, request.Desired); err != nil {
			return ledger.Change{}, wrap(CodeInvalid, "Desired state must be valid JSON.", err)
		}
		request.Desired = append(json.RawMessage(nil), compact.Bytes()...)
	} else if request.Action == ledger.ChangeDelete {
		request.Desired = nil
	}
	requestFingerprint, err := ledger.Hash(struct {
		Unit    string              `json:"unit"`
		Action  ledger.ChangeAction `json:"action"`
		Desired json.RawMessage     `json:"desired,omitempty"`
	}{request.Unit, request.Action, request.Desired})
	if err != nil {
		return ledger.Change{}, wrap(CodeInvalid, "Could not canonicalize the Change.", err)
	}
	existing, err := engine.repository.FindChangeByIdempotencyKey(ctx, ledgerID, request.IdempotencyKey)
	if err == nil {
		if existing.RequestFingerprint != requestFingerprint {
			return ledger.Change{}, wrap(CodeConflict, "Idempotency key was already used for a different Change.", repository.ErrIdempotencyConflict)
		}
		return existing, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return ledger.Change{}, wrap(CodeInternal, "Could not check Change idempotency.", err)
	}
	id, err := ledger.NewID("chg")
	if err != nil {
		return ledger.Change{}, wrap(CodeInternal, "Could not create the Change.", err)
	}
	value := ledger.Change{ID: id, LedgerID: ledgerID, Unit: request.Unit, Action: request.Action, Desired: request.Desired, DesiredFingerprint: ledger.Fingerprint(request.Desired), IdempotencyKey: request.IdempotencyKey, RequestFingerprint: requestFingerprint, Status: ledger.ChangeReady, CreatedAt: time.Now().UTC()}
	if err := ledger.ValidateChange(value); err != nil {
		return ledger.Change{}, wrap(CodeInvalid, err.Error(), err)
	}
	if len(value.Desired) > 0 {
		if _, err := engine.repository.WriteObject(ctx, "VALUE", value.Desired); err != nil {
			return ledger.Change{}, wrap(CodeInternal, "Could not durably store the proposed value.", err)
		}
	}
	if err := engine.repository.InsertChange(ctx, &value); err != nil {
		existing, lookupErr := engine.repository.FindChangeByIdempotencyKey(ctx, ledgerID, request.IdempotencyKey)
		if lookupErr == nil && existing.RequestFingerprint == requestFingerprint {
			return existing, nil
		}
		return ledger.Change{}, wrap(CodeInternal, "Could not durably accept the Change.", err)
	}
	engine.metrics.ChangeAccepted()
	engine.publish(EventChangeAccepted, ledgerID, value.ID)
	return value, nil
}
