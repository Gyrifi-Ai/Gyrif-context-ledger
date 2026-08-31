package ledger

import "time"

type Release struct {
	ID           string    `json:"id"`
	LedgerID     string    `json:"ledgerId"`
	ProposalID   string    `json:"proposalId"`
	ProposalHash string    `json:"proposalHash"`
	ParentID     string    `json:"parentId,omitempty"`
	Hash         string    `json:"hash"`
	CreatedAt    time.Time `json:"createdAt"`
}

type ReleaseIntentStatus string

const (
	IntentReady            ReleaseIntentStatus = "READY"
	IntentApplying         ReleaseIntentStatus = "APPLYING"
	IntentVerifying        ReleaseIntentStatus = "VERIFYING"
	IntentFinalized        ReleaseIntentStatus = "FINALIZED"
	IntentRecoveryRequired ReleaseIntentStatus = "RECOVERY_REQUIRED"
	IntentAbandoned        ReleaseIntentStatus = "ABANDONED"
)

type ReleaseIntent struct {
	ID             string              `json:"id"`
	LedgerID       string              `json:"ledgerId"`
	ProposalID     string              `json:"proposalId"`
	ProposalHash   string              `json:"proposalHash"`
	ParentID       string              `json:"parentId,omitempty"`
	Status         ReleaseIntentStatus `json:"status"`
	Plan           []byte              `json:"plan"`
	CreatedAt      time.Time           `json:"createdAt"`
	Resolution     string              `json:"resolution,omitempty"`
	ResolutionNote string              `json:"resolutionNote,omitempty"`
	ResolvedAt     *time.Time          `json:"resolvedAt,omitempty"`
}
