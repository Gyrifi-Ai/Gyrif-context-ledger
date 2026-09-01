package ledger

import (
	"encoding/json"
	"time"
)

type ChangeAction string

const (
	ChangePut    ChangeAction = "PUT"
	ChangeDelete ChangeAction = "DELETE"
)

type ChangeStatus string

const (
	ChangeAccepted  ChangeStatus = "ACCEPTED"
	ChangeReady     ChangeStatus = "READY"
	ChangeInvalid   ChangeStatus = "INVALID"
	ChangeReleased  ChangeStatus = "RELEASED"
	ChangeWithdrawn ChangeStatus = "WITHDRAWN"
)

type Change struct {
	ID                  string          `json:"id"`
	LedgerID            string          `json:"ledgerId"`
	Sequence            int64           `json:"sequence"`
	Unit                string          `json:"unit"`
	Action              ChangeAction    `json:"action"`
	Desired             json.RawMessage `json:"desired,omitempty"`
	BaseFingerprint     string          `json:"baseFingerprint"`
	DesiredFingerprint  string          `json:"desiredFingerprint"`
	IdempotencyKey      string          `json:"-"`
	RequestFingerprint  string          `json:"-"`
	Status              ChangeStatus    `json:"status"`
	InvalidReason       string          `json:"invalidReason,omitempty"`
	Noop                bool            `json:"noop"`
	Stalled             bool            `json:"stalled"`
	PreparationAttempts int             `json:"-"`
	CreatedAt           time.Time       `json:"createdAt"`
}

type PreparationOutcome struct {
	Status          ChangeStatus
	BaseFingerprint string
	Noop            bool
}

func PrepareChangeOutcome(change Change, observedFingerprint string, exists bool) PreparationOutcome {
	noop := (change.Action == ChangePut && exists && observedFingerprint == change.DesiredFingerprint) || (change.Action == ChangeDelete && !exists)
	return PreparationOutcome{Status: ChangeReady, BaseFingerprint: observedFingerprint, Noop: noop}
}
