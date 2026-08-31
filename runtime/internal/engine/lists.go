package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
)

const (
	DefaultListLimit = 50
	MaxListLimit     = 200
)

type ListRequest struct {
	Limit  int
	Cursor string
	Status string
	Action string
}

type ListPage[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
}

var (
	changeStatuses   = []string{string(ledger.ChangeAccepted), string(ledger.ChangeReady), string(ledger.ChangeInvalid), string(ledger.ChangeReleased)}
	proposalStatuses = []string{string(ledger.ProposalDraft), string(ledger.ProposalReviewed), string(ledger.ProposalApproved), string(ledger.ProposalReleased), string(ledger.ProposalBlocked), string(ledger.ProposalCancelled)}
	changeActions    = []string{string(ledger.ChangePut), string(ledger.ChangeDelete)}
)

func listOptions(request ListRequest, allowedStatuses, allowedActions []string) (repository.ListOptions, error) {
	limit := request.Limit
	if limit == 0 {
		limit = DefaultListLimit
	}
	if limit < 1 || limit > MaxListLimit {
		return repository.ListOptions{}, wrap(CodeInvalid, "Limit must be between 1 and 200.", ledger.ErrInvalid)
	}
	options := repository.ListOptions{Limit: limit}
	if request.Cursor != "" {
		cursor, err := repository.DecodeCursor(request.Cursor)
		if err != nil {
			return repository.ListOptions{}, wrap(CodeInvalid, "The cursor is not valid.", err)
		}
		options.Cursor = &cursor
	}
	if request.Status != "" {
		if !contains(allowedStatuses, request.Status) {
			return repository.ListOptions{}, wrap(CodeInvalid, fmt.Sprintf("Status must be one of: %s.", strings.Join(allowedStatuses, ", ")), ledger.ErrInvalid)
		}
		status := request.Status
		options.Status = &status
	}
	if request.Action != "" {
		if !contains(allowedActions, request.Action) {
			return repository.ListOptions{}, wrap(CodeInvalid, fmt.Sprintf("Action must be one of: %s.", strings.Join(allowedActions, ", ")), ledger.ErrInvalid)
		}
		action := request.Action
		options.Action = &action
	}
	return options, nil
}

func contains(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func (engine *Engine) ListLedgers(ctx context.Context, request ListRequest) (ListPage[ledger.Ledger], error) {
	options, err := listOptions(request, nil, nil)
	if err != nil {
		return ListPage[ledger.Ledger]{}, err
	}
	page, err := engine.repository.ListLedgers(ctx, options)
	if err != nil {
		return ListPage[ledger.Ledger]{}, wrap(CodeInternal, "Could not load ledgers.", err)
	}
	result := ListPage[ledger.Ledger]{Items: page.Items}
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		result.NextCursor = repository.EncodeCursor(last.CreatedAt, last.ID)
	}
	return result, nil
}

func (engine *Engine) ListChanges(ctx context.Context, ledgerID string, request ListRequest) (ListPage[ledger.Change], error) {
	options, err := listOptions(request, changeStatuses, changeActions)
	if err != nil {
		return ListPage[ledger.Change]{}, err
	}
	page, err := engine.repository.ListChanges(ctx, ledgerID, options)
	if err != nil {
		return ListPage[ledger.Change]{}, wrap(CodeInternal, "Could not load Changes.", err)
	}
	result := ListPage[ledger.Change]{Items: page.Items}
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		result.NextCursor = repository.EncodeCursor(last.CreatedAt, last.ID)
	}
	return result, nil
}

func (engine *Engine) ListProposals(ctx context.Context, ledgerID string, request ListRequest) (ListPage[ledger.Proposal], error) {
	options, err := listOptions(request, proposalStatuses, nil)
	if err != nil {
		return ListPage[ledger.Proposal]{}, err
	}
	page, err := engine.repository.ListProposals(ctx, ledgerID, options)
	if err != nil {
		return ListPage[ledger.Proposal]{}, wrap(CodeInternal, "Could not load Proposals.", err)
	}
	result := ListPage[ledger.Proposal]{Items: page.Items}
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		result.NextCursor = repository.EncodeCursor(last.CreatedAt, last.ID)
	}
	return result, nil
}

func (engine *Engine) ListReleases(ctx context.Context, ledgerID string, request ListRequest) (ListPage[ledger.Release], error) {
	options, err := listOptions(request, nil, nil)
	if err != nil {
		return ListPage[ledger.Release]{}, err
	}
	page, err := engine.repository.ListReleases(ctx, ledgerID, options)
	if err != nil {
		return ListPage[ledger.Release]{}, wrap(CodeInternal, "Could not load Releases.", err)
	}
	result := ListPage[ledger.Release]{Items: page.Items}
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		result.NextCursor = repository.EncodeCursor(last.CreatedAt, last.ID)
	}
	return result, nil
}
