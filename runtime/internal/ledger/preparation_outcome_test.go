package ledger

import "testing"

func TestPrepareChangeOutcome(t *testing.T) {
	tests := []struct {
		name     string
		change   Change
		observed string
		exists   bool
		noop     bool
		base     string
	}{
		{name: "put absent", change: Change{Action: ChangePut, DesiredFingerprint: "sha256:desired"}},
		{name: "put differs", change: Change{Action: ChangePut, DesiredFingerprint: "sha256:desired"}, observed: "sha256:base", exists: true, base: "sha256:base"},
		{name: "put identical", change: Change{Action: ChangePut, DesiredFingerprint: "sha256:same"}, observed: "sha256:same", exists: true, noop: true, base: "sha256:same"},
		{name: "delete absent", change: Change{Action: ChangeDelete}, noop: true},
		{name: "delete present", change: Change{Action: ChangeDelete}, observed: "sha256:base", exists: true, base: "sha256:base"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outcome := PrepareChangeOutcome(test.change, test.observed, test.exists)
			if outcome.Status != ChangeReady || outcome.Noop != test.noop || outcome.BaseFingerprint != test.base {
				t.Fatalf("outcome = %#v", outcome)
			}
		})
	}
}
