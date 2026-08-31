package repository

import (
	"context"
	"errors"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different request")
	ErrChangeClaimed       = errors.New("change is already claimed by an active proposal")
)

type Repository interface {
	CreateLedger(context.Context, ledger.Ledger) error
	ListLedgers(context.Context) ([]ledger.Ledger, error)
	FindChangeByIdempotencyKey(context.Context, string, string) (ledger.Change, error)
	InsertChange(context.Context, *ledger.Change) error
	ListChanges(context.Context, string) ([]ledger.Change, error)
	LoadChanges(context.Context, string, []string) ([]ledger.Change, error)
	InsertProposal(context.Context, ledger.Proposal) error
	LoadProposal(context.Context, string, string) (ledger.Proposal, error)
	ListProposals(context.Context, string) ([]ledger.Proposal, error)
	SaveCheckResult(context.Context, ledger.CheckResult) error
	ListCheckResults(context.Context, string) ([]ledger.CheckResult, error)
	HasPassingCheck(context.Context, string, string) (bool, error)
	SaveApproval(context.Context, ledger.Approval) error
	ListApprovals(context.Context, string) ([]ledger.Approval, error)
	HasApproval(context.Context, string, string) (bool, error)
	CurrentHead(context.Context, string) (ledger.Head, error)
	SaveReleaseIntent(context.Context, ledger.ReleaseIntent) error
	UpdateReleaseIntent(context.Context, string, ledger.ReleaseIntentStatus) error
	ListUnfinishedReleaseIntents(context.Context) ([]ledger.ReleaseIntent, error)
	LoadReleaseIntentForProposal(context.Context, string) (ledger.ReleaseIntent, error)
	FinalizeRelease(context.Context, ledger.ReleaseIntent, ledger.Release) error
	ListReleases(context.Context, string) ([]ledger.Release, error)
	WriteObject(context.Context, string, []byte) (string, error)
	ReadObject(context.Context, string) ([]byte, error)
	Close() error
}
