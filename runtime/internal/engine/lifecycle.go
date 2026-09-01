package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
)

func (engine *Engine) ensureLedgerWritable(ctx context.Context, ledgerID string) error {
	value, err := engine.repository.LoadLedger(ctx, ledgerID)
	if errors.Is(err, repository.ErrNotFound) {
		return wrap(CodeNotFound, "Ledger was not found.", err)
	}
	if err != nil {
		return wrap(CodeInternal, "Could not load Ledger.", err)
	}
	if value.ArchivedAt != nil {
		return wrap(CodeConflict, "This Ledger is archived.", repository.ErrLedgerArchived)
	}
	return nil
}

func (engine *Engine) WithdrawChange(ctx context.Context, ledgerID, changeID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return wrap(CodeInvalid, "Withdrawal reason is required.", ledger.ErrInvalid)
	}
	changed, err := engine.repository.WithdrawChange(ctx, ledgerID, changeID, reason, time.Now().UTC())
	if err != nil {
		var claimed *repository.ChangeClaimError
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return wrap(CodeNotFound, "Change was not found.", err)
		case errors.As(err, &claimed):
			return wrap(CodeConflict, fmt.Sprintf("This Change belongs to Proposal %s. Cancel the Proposal first.", claimed.ProposalID), err)
		case errors.Is(err, ledger.ErrConflict):
			return wrap(CodeConflict, "A released Change is part of the audit trail and cannot be withdrawn.", err)
		default:
			return wrap(CodeInternal, "Could not withdraw Change.", err)
		}
	}
	if changed {
		engine.publish(EventChangeWithdrawn, ledgerID, changeID)
	}
	return nil
}

func (engine *Engine) ArchiveLedger(ctx context.Context, ledgerID string) error {
	engine.releaseMu.Lock()
	defer engine.releaseMu.Unlock()
	changed, err := engine.repository.ArchiveLedger(ctx, ledgerID, time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return wrap(CodeNotFound, "Ledger was not found.", err)
		case errors.Is(err, repository.ErrLedgerWorkInFlight):
			return wrap(CodeConflict, "This Ledger has work in flight and cannot be archived.", err)
		default:
			return wrap(CodeInternal, "Could not archive Ledger.", err)
		}
	}
	if changed {
		engine.publish(EventLedgerArchived, ledgerID, ledgerID)
	}
	return nil
}

func (engine *Engine) UnarchiveLedger(ctx context.Context, ledgerID string) error {
	changed, err := engine.repository.UnarchiveLedger(ctx, ledgerID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return wrap(CodeNotFound, "Ledger was not found.", err)
		}
		return wrap(CodeInternal, "Could not unarchive Ledger.", err)
	}
	if changed {
		engine.publish(EventLedgerUnarchived, ledgerID, ledgerID)
	}
	return nil
}
