package knowledge

import (
	"testing"
	"time"
)

func testPolicy() VotingPolicy {
	return VotingPolicy{
		Enabled:       true,
		MinPoolSize:   3,
		QuorumYes:     2,
		QuorumNo:      2,
		Timeout:       2 * time.Minute,
		AllowSelfVote: false,
	}
}

func TestVotingPolicyValidate(t *testing.T) {
	if err := testPolicy().Validate(); err != nil {
		t.Fatalf("expected valid policy, got %v", err)
	}
	bad := testPolicy()
	bad.Timeout = 0
	if err := bad.Validate(); err == nil {
		t.Fatal("expected timeout validation error")
	}
}

func TestEvaluateQuorum_ApprovedByYes(t *testing.T) {
	now := time.Now()
	decision := EvaluateQuorum("proposer", 4, map[string]string{
		"claw-a": "yes",
		"claw-b": "yes",
	}, now.Add(-30*time.Second), now, testPolicy())
	if decision.Status != VoteStatusApproved {
		t.Fatalf("expected approved, got %+v", decision)
	}
	if decision.Yes != 2 || decision.No != 0 {
		t.Fatalf("unexpected tally: %+v", decision)
	}
}

func TestEvaluateQuorum_RejectedByNo(t *testing.T) {
	now := time.Now()
	decision := EvaluateQuorum("proposer", 4, map[string]string{
		"claw-a": "no",
		"claw-b": "no",
		"claw-c": "yes",
	}, now.Add(-30*time.Second), now, testPolicy())
	if decision.Status != VoteStatusRejected {
		t.Fatalf("expected rejected, got %+v", decision)
	}
	if decision.Yes != 1 || decision.No != 2 {
		t.Fatalf("unexpected tally: %+v", decision)
	}
}

func TestEvaluateQuorum_ExpiredOnTimeout(t *testing.T) {
	now := time.Now()
	decision := EvaluateQuorum("proposer", 4, map[string]string{
		"claw-a": "yes",
	}, now.Add(-10*time.Minute), now, testPolicy())
	if decision.Status != VoteStatusExpired {
		t.Fatalf("expected expired, got %+v", decision)
	}
}

func TestEvaluateQuorum_PendingWithoutQuorum(t *testing.T) {
	now := time.Now()
	decision := EvaluateQuorum("proposer", 4, map[string]string{
		"claw-a": "yes",
	}, now.Add(-20*time.Second), now, testPolicy())
	if decision.Status != VoteStatusPending {
		t.Fatalf("expected pending, got %+v", decision)
	}
}

func TestEvaluateQuorum_PoolBelowMinSkipsVoting(t *testing.T) {
	now := time.Now()
	decision := EvaluateQuorum("proposer", 2, map[string]string{}, now, now, testPolicy())
	if decision.Status != VoteStatusApproved {
		t.Fatalf("expected approved due to small pool, got %+v", decision)
	}
}

func TestEvaluateQuorum_SelfVoteExcludedByDefault(t *testing.T) {
	now := time.Now()
	decision := EvaluateQuorum("proposer", 4, map[string]string{
		"proposer": "yes",
		"claw-a":   "yes",
	}, now.Add(-30*time.Second), now, testPolicy())
	if decision.Yes != 1 {
		t.Fatalf("expected proposer vote excluded, got %+v", decision)
	}
	if decision.Status != VoteStatusPending {
		t.Fatalf("expected pending without second non-self yes, got %+v", decision)
	}
}

func TestEvaluateQuorum_SelfVoteAllowedWhenConfigured(t *testing.T) {
	now := time.Now()
	p := testPolicy()
	p.AllowSelfVote = true
	decision := EvaluateQuorum("proposer", 4, map[string]string{
		"proposer": "yes",
		"claw-a":   "yes",
	}, now.Add(-30*time.Second), now, p)
	if decision.Status != VoteStatusApproved {
		t.Fatalf("expected approved with self vote allowed, got %+v", decision)
	}
}

func TestEvaluateQuorum_InvalidPolicyRejected(t *testing.T) {
	now := time.Now()
	p := testPolicy()
	p.MinPoolSize = 0
	decision := EvaluateQuorum("proposer", 4, nil, now, now, p)
	if decision.Status != VoteStatusRejected {
		t.Fatalf("expected rejected for invalid policy, got %+v", decision)
	}
}

func TestEvaluateQuorum_DisabledPolicyApproves(t *testing.T) {
	now := time.Now()
	p := testPolicy()
	p.Enabled = false
	decision := EvaluateQuorum("proposer", 10, nil, now, now, p)
	if decision.Status != VoteStatusApproved {
		t.Fatalf("expected approved with voting disabled, got %+v", decision)
	}
}
