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
	ChangeAccepted ChangeStatus = "ACCEPTED"
	ChangeReady    ChangeStatus = "READY"
	ChangeInvalid  ChangeStatus = "INVALID"
	ChangeReleased ChangeStatus = "RELEASED"
)

type Change struct {
	ID                 string          `json:"id"`
	LedgerID           string          `json:"ledgerId"`
	Sequence           int64           `json:"sequence"`
	Unit               string          `json:"unit"`
	Action             ChangeAction    `json:"action"`
	Desired            json.RawMessage `json:"desired,omitempty"`
	BaseFingerprint    string          `json:"baseFingerprint"`
	DesiredFingerprint string          `json:"desiredFingerprint"`
	IdempotencyKey     string          `json:"-"`
	RequestFingerprint string          `json:"-"`
	Status             ChangeStatus    `json:"status"`
	CreatedAt          time.Time       `json:"createdAt"`
}
