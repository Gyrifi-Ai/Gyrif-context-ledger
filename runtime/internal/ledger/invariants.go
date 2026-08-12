package ledger

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalid         = errors.New("invalid governance state")
	ErrConflict        = errors.New("governance conflict")
	ErrStaleEvidence   = errors.New("evidence does not match current proposal")
	ErrReleaseNotReady = errors.New("proposal is not ready to release")
)

func ValidateChange(change Change) error {
	if strings.TrimSpace(change.LedgerID) == "" || strings.TrimSpace(change.Unit) == "" {
		return fmt.Errorf("%w: ledger and logical unit are required", ErrInvalid)
	}
	if change.Action != ChangePut && change.Action != ChangeDelete {
		return fmt.Errorf("%w: action must be PUT or DELETE", ErrInvalid)
	}
	if change.Action == ChangePut && len(change.Desired) == 0 {
		return fmt.Errorf("%w: PUT requires desired state", ErrInvalid)
	}
	return nil
}

func ProposalHash(proposal Proposal) (string, error) {
	return Hash(struct {
		Version       int      `json:"version"`
		LedgerID      string   `json:"ledgerId"`
		BaseReleaseID string   `json:"baseReleaseId"`
		ChangeIDs     []string `json:"changeIds"`
	}{1, proposal.LedgerID, proposal.BaseReleaseID, proposal.ChangeIDs})
}

func ReleaseHash(release Release) (string, error) {
	return Hash(struct {
		Version      int    `json:"version"`
		LedgerID     string `json:"ledgerId"`
		ProposalID   string `json:"proposalId"`
		ParentID     string `json:"parentId"`
		ProposalHash string `json:"proposalHash"`
	}{1, release.LedgerID, release.ProposalID, release.ParentID, release.ProposalHash})
}

func ApprovalIsCurrent(approval Approval, proposal Proposal) bool {
	return approval.ProposalID == proposal.ID && approval.ProposalHash == proposal.Hash
}

func ValidateReleaseParent(current Head, intent ReleaseIntent) error {
	if current.LedgerID != intent.LedgerID || current.ReleaseID != intent.ParentID {
		return fmt.Errorf("%w: proposal base no longer matches HEAD", ErrConflict)
	}
	return nil
}
