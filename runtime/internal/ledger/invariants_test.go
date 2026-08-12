package ledger

import "testing"

func TestProposalHashIsDeterministicAndOrderSensitive(t *testing.T) {
	first := Proposal{LedgerID: "ledger", BaseReleaseID: "release", ChangeIDs: []string{"a", "b"}}
	second := first
	firstHash, err := ProposalHash(first)
	if err != nil {
		t.Fatal(err)
	}
	secondHash, err := ProposalHash(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstHash != secondHash {
		t.Fatalf("same proposal produced different hashes")
	}
	second.ChangeIDs = []string{"b", "a"}
	orderedHash, err := ProposalHash(second)
	if err != nil {
		t.Fatal(err)
	}
	if orderedHash == firstHash {
		t.Fatalf("change order must affect proposal identity")
	}
}
func TestApprovalIsBoundToProposalHash(t *testing.T) {
	proposal := Proposal{ID: "pr", Hash: "new"}
	if ApprovalIsCurrent(Approval{ProposalID: "pr", ProposalHash: "old"}, proposal) {
		t.Fatal("stale approval was accepted")
	}
	if !ApprovalIsCurrent(Approval{ProposalID: "pr", ProposalHash: "new"}, proposal) {
		t.Fatal("current approval was rejected")
	}
}
