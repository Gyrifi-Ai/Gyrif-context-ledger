package repository

import (
	"context"
	"errors"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
)

var (
	ErrNotFound                 = errors.New("not found")
	ErrIdempotencyConflict      = errors.New("idempotency key reused with different request")
	ErrChangeClaimed            = errors.New("change is already claimed by an active proposal")
	ErrProposalAlreadyCancelled = errors.New("proposal is already cancelled")
	ErrProposalNotDraft         = errors.New("proposal is not a draft")
	ErrProposalReleased         = errors.New("proposal is released")
	ErrProposalReleaseStarted   = errors.New("proposal release has started")
)

type ListOptions struct {
	Limit  int
	Cursor *Cursor
	Status *string
	Action *string
}

type Page[T any] struct {
	Items   []T
	HasMore bool
}

type Repository interface {
	CreateLedger(context.Context, ledger.Ledger) error
	ListLedgers(context.Context, ListOptions) (Page[ledger.Ledger], error)
	FindChangeByIdempotencyKey(context.Context, string, string) (ledger.Change, error)
	InsertChange(context.Context, *ledger.Change) error
	ListChanges(context.Context, string, ListOptions) (Page[ledger.Change], error)
	LoadChanges(context.Context, string, []string) ([]ledger.Change, error)
	InsertProposal(context.Context, ledger.Proposal) error
	LoadProposal(context.Context, string, string) (ledger.Proposal, error)
	ListProposals(context.Context, string, ListOptions) (Page[ledger.Proposal], error)
	CancelProposal(context.Context, string, string) error
	HasReleaseIntent(context.Context, string) (bool, error)
	SaveCheckResult(context.Context, ledger.CheckResult) error
	ListCheckResults(context.Context, string) ([]ledger.CheckResult, error)
	HasPassingCheck(context.Context, string, string) (bool, error)
	SaveApproval(context.Context, ledger.Approval) error
	ListApprovals(context.Context, string) ([]ledger.Approval, error)
	HasApproval(context.Context, string, string) (bool, error)
	CurrentHead(context.Context, string) (ledger.Head, error)
	SaveReleaseIntent(context.Context, ledger.ReleaseIntent) error
	UpdateReleaseIntent(context.Context, string, ledger.ReleaseIntentStatus) error
	ResolveReleaseIntent(context.Context, string, string, time.Time) error
	LoadReleaseIntent(context.Context, string) (ledger.ReleaseIntent, error)
	ListReleaseIntentsForLedger(context.Context, string, *ledger.ReleaseIntentStatus) ([]ledger.ReleaseIntent, error)
	ListUnfinishedReleaseIntents(context.Context) ([]ledger.ReleaseIntent, error)
	LoadReleaseIntentForProposal(context.Context, string) (ledger.ReleaseIntent, error)
	FinalizeRelease(context.Context, ledger.ReleaseIntent, ledger.Release) error
	ListReleases(context.Context, string, ListOptions) (Page[ledger.Release], error)
	WriteObject(context.Context, string, []byte) (string, error)
	ReadObject(context.Context, string) ([]byte, error)
	Close() error
}
