package knowledge

import (
	"testing"
	"time"
)

func TestEnvelopeValidateBase(t *testing.T) {
	env := Envelope{
		SchemaVersion:  CurrentSchemaVersion,
		Type:           TypeProposal,
		TraceID:        "trace-1",
		Timestamp:      time.Now(),
		IdempotencyKey: "idem-1",
		ClawID:         "claw-a",
		InstanceID:     "inst-a",
	}
	if err := env.ValidateBase(); err != nil {
		t.Fatalf("validate base: %v", err)
	}
}

func TestEnvelopeValidateBaseRejectsInvalid(t *testing.T) {
	env := Envelope{Type: "invalid"}
	if err := env.ValidateBase(); err == nil {
		t.Fatal("expected error for invalid envelope")
	}
}

func TestProposalPayloadValidate(t *testing.T) {
	p := ProposalPayload{
		ProposalID: "p1",
		Group:      "g1",
		Statement:  "Use runbook v2",
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("proposal validate: %v", err)
	}
	if err := (ProposalPayload{}).Validate(); err == nil {
		t.Fatal("expected error for empty proposal payload")
	}
}

func TestVotePayloadValidate(t *testing.T) {
	v := VotePayload{ProposalID: "p1", Vote: "yes"}
	if err := v.Validate(); err != nil {
		t.Fatalf("vote validate: %v", err)
	}
	if err := (VotePayload{ProposalID: "p1", Vote: "maybe"}).Validate(); err == nil {
		t.Fatal("expected error for invalid vote")
	}
}

func TestDecisionPayloadValidate(t *testing.T) {
	d := DecisionPayload{ProposalID: "p1", Outcome: "approved", Yes: 2, No: 1}
	if err := d.Validate(); err != nil {
		t.Fatalf("decision validate: %v", err)
	}
	if err := (DecisionPayload{ProposalID: "p1", Outcome: "won", Yes: 1, No: 0}).Validate(); err == nil {
		t.Fatal("expected error for invalid outcome")
	}
}

func TestFactPayloadValidate(t *testing.T) {
	f := FactPayload{
		FactID:    "f1",
		Group:     "g1",
		Subject:   "service",
		Predicate: "runbook",
		Object:    "v2",
		Version:   1,
		Source:    "decision:d1",
	}
	if err := f.Validate(); err != nil {
		t.Fatalf("fact validate: %v", err)
	}
	if err := (FactPayload{FactID: "f1", Group: "g1", Version: 0, Source: "x"}).Validate(); err == nil {
		t.Fatal("expected error for invalid fact payload")
	}
}
