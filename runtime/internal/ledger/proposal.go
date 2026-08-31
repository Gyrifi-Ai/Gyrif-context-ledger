package ledger

import "time"

type ProposalStatus string

const (
	ProposalDraft     ProposalStatus = "DRAFT"
	ProposalReviewed  ProposalStatus = "REVIEWED"
	ProposalApproved  ProposalStatus = "APPROVED"
	ProposalReleased  ProposalStatus = "RELEASED"
	ProposalBlocked   ProposalStatus = "BLOCKED"
	ProposalCancelled ProposalStatus = "CANCELLED"
)

type Proposal struct {
	ID            string         `json:"id"`
	LedgerID      string         `json:"ledgerId"`
	Title         string         `json:"title"`
	BaseReleaseID string         `json:"baseReleaseId,omitempty"`
	Hash          string         `json:"hash"`
	Status        ProposalStatus `json:"status"`
	ChangeIDs     []string       `json:"changeIds"`
	CreatedAt     time.Time      `json:"createdAt"`
}

type CheckResult struct {
	ID           string    `json:"id"`
	ProposalID   string    `json:"proposalId"`
	ProposalHash string    `json:"proposalHash"`
	Kind         string    `json:"kind"`
	Passed       bool      `json:"passed"`
	Summary      string    `json:"summary"`
	Evidence     []byte    `json:"evidence,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Approval struct {
	ID           string    `json:"id"`
	ProposalID   string    `json:"proposalId"`
	ProposalHash string    `json:"proposalHash"`
	Actor        string    `json:"actor"`
	CreatedAt    time.Time `json:"createdAt"`
}
