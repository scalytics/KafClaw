package knowledge

import (
	"fmt"
	"strings"
	"time"
)

const (
	VoteStatusPending  = "pending"
	VoteStatusApproved = "approved"
	VoteStatusRejected = "rejected"
	VoteStatusExpired  = "expired"
)

type VotingPolicy struct {
	Enabled       bool
	MinPoolSize   int
	QuorumYes     int
	QuorumNo      int
	Timeout       time.Duration
	AllowSelfVote bool
}

type VoteDecision struct {
	Status string
	Yes    int
	No     int
	Reason string
}

func (p VotingPolicy) Validate() error {
	if p.MinPoolSize <= 0 {
		return fmt.Errorf("min pool size must be > 0")
	}
	if p.QuorumYes <= 0 || p.QuorumNo <= 0 {
		return fmt.Errorf("quorum yes/no must be > 0")
	}
	if p.Timeout <= 0 {
		return fmt.Errorf("timeout must be > 0")
	}
	return nil
}

// EvaluateQuorum applies proposal-voting decision rules:
// - voting activates only when poolSize >= MinPoolSize
// - one vote per clawId (map key)
// - self vote excluded unless AllowSelfVote=true
// - approved: yes >= QuorumYes and yes > no
// - rejected: no >= QuorumNo
// - expired: timeout elapsed without quorum
func EvaluateQuorum(
	proposerClawID string,
	poolSize int,
	votes map[string]string,
	createdAt time.Time,
	now time.Time,
	policy VotingPolicy,
) VoteDecision {
	if err := policy.Validate(); err != nil {
		return VoteDecision{Status: VoteStatusRejected, Reason: err.Error()}
	}
	if !policy.Enabled {
		return VoteDecision{Status: VoteStatusApproved, Reason: "voting disabled"}
	}
	if poolSize < policy.MinPoolSize {
		return VoteDecision{Status: VoteStatusApproved, Reason: "pool below min size"}
	}

	yes, no := tallyVotes(votes, proposerClawID, policy.AllowSelfVote)
	if yes >= policy.QuorumYes && yes > no {
		return VoteDecision{Status: VoteStatusApproved, Yes: yes, No: no}
	}
	if no >= policy.QuorumNo {
		return VoteDecision{Status: VoteStatusRejected, Yes: yes, No: no}
	}
	if !createdAt.IsZero() && now.Sub(createdAt) >= policy.Timeout {
		return VoteDecision{Status: VoteStatusExpired, Yes: yes, No: no, Reason: "voting timeout"}
	}
	return VoteDecision{Status: VoteStatusPending, Yes: yes, No: no}
}

func tallyVotes(votes map[string]string, proposerClawID string, allowSelf bool) (yes int, no int) {
	for clawID, v := range votes {
		if !allowSelf && strings.EqualFold(strings.TrimSpace(clawID), strings.TrimSpace(proposerClawID)) {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "yes":
			yes++
		case "no":
			no++
		}
	}
	return yes, no
}
