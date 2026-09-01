package ledger

import "time"

type Ledger struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"createdAt"`
	ArchivedAt  *time.Time `json:"archivedAt,omitempty"`
}

type Head struct {
	LedgerID  string `json:"ledgerId"`
	ReleaseID string `json:"releaseId,omitempty"`
}
