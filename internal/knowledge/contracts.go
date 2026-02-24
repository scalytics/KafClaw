package knowledge

import (
	"fmt"
	"strings"
	"time"
)

const CurrentSchemaVersion = "v1"

const (
	TypeCapabilities = "capabilities"
	TypePresence     = "presence"
	TypeProposal     = "proposal"
	TypeVote         = "vote"
	TypeDecision     = "decision"
	TypeFact         = "fact"
)

type Envelope struct {
	SchemaVersion  string    `json:"schemaVersion"`
	Type           string    `json:"type"`
	TraceID        string    `json:"traceId"`
	Timestamp      time.Time `json:"timestamp"`
	IdempotencyKey string    `json:"idempotencyKey"`
	ClawID         string    `json:"clawId"`
	InstanceID     string    `json:"instanceId"`
	Payload        any       `json:"payload"`
}

func (e Envelope) ValidateBase() error {
	if strings.TrimSpace(e.SchemaVersion) == "" {
		return fmt.Errorf("schemaVersion is required")
	}
	if strings.TrimSpace(e.Type) == "" {
		return fmt.Errorf("type is required")
	}
	if strings.TrimSpace(e.TraceID) == "" {
		return fmt.Errorf("traceId is required")
	}
	if e.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}
	if strings.TrimSpace(e.IdempotencyKey) == "" {
		return fmt.Errorf("idempotencyKey is required")
	}
	if strings.TrimSpace(e.ClawID) == "" {
		return fmt.Errorf("clawId is required")
	}
	if strings.TrimSpace(e.InstanceID) == "" {
		return fmt.Errorf("instanceId is required")
	}
	switch e.Type {
	case TypeCapabilities, TypePresence, TypeProposal, TypeVote, TypeDecision, TypeFact:
		return nil
	default:
		return fmt.Errorf("unsupported type: %s", e.Type)
	}
}

type ProposalPayload struct {
	ProposalID string   `json:"proposalId"`
	Group      string   `json:"group"`
	Title      string   `json:"title"`
	Statement  string   `json:"statement"`
	Tags       []string `json:"tags,omitempty"`
}

func (p ProposalPayload) Validate() error {
	if strings.TrimSpace(p.ProposalID) == "" {
		return fmt.Errorf("proposalId is required")
	}
	if strings.TrimSpace(p.Group) == "" {
		return fmt.Errorf("group is required")
	}
	if strings.TrimSpace(p.Statement) == "" {
		return fmt.Errorf("statement is required")
	}
	return nil
}

type VotePayload struct {
	ProposalID string `json:"proposalId"`
	Vote       string `json:"vote"` // yes|no
	Reason     string `json:"reason,omitempty"`
}

func (p VotePayload) Validate() error {
	if strings.TrimSpace(p.ProposalID) == "" {
		return fmt.Errorf("proposalId is required")
	}
	switch strings.ToLower(strings.TrimSpace(p.Vote)) {
	case "yes", "no":
		return nil
	default:
		return fmt.Errorf("vote must be yes|no")
	}
}

type DecisionPayload struct {
	ProposalID string `json:"proposalId"`
	Outcome    string `json:"outcome"` // approved|rejected|expired
	Yes        int    `json:"yes"`
	No         int    `json:"no"`
	Reason     string `json:"reason,omitempty"`
}

func (p DecisionPayload) Validate() error {
	if strings.TrimSpace(p.ProposalID) == "" {
		return fmt.Errorf("proposalId is required")
	}
	switch strings.ToLower(strings.TrimSpace(p.Outcome)) {
	case "approved", "rejected", "expired":
	default:
		return fmt.Errorf("outcome must be approved|rejected|expired")
	}
	if p.Yes < 0 || p.No < 0 {
		return fmt.Errorf("yes/no must be >= 0")
	}
	return nil
}

type FactPayload struct {
	FactID      string   `json:"factId"`
	Group       string   `json:"group"`
	Subject     string   `json:"subject"`
	Predicate   string   `json:"predicate"`
	Object      string   `json:"object"`
	Version     int      `json:"version"`
	Source      string   `json:"source"`
	ProposalID  string   `json:"proposalId,omitempty"`
	DecisionID  string   `json:"decisionId,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	PublishedAt string   `json:"publishedAt,omitempty"`
}

func (p FactPayload) Validate() error {
	if strings.TrimSpace(p.FactID) == "" {
		return fmt.Errorf("factId is required")
	}
	if strings.TrimSpace(p.Group) == "" {
		return fmt.Errorf("group is required")
	}
	if strings.TrimSpace(p.Subject) == "" || strings.TrimSpace(p.Predicate) == "" || strings.TrimSpace(p.Object) == "" {
		return fmt.Errorf("subject/predicate/object are required")
	}
	if p.Version <= 0 {
		return fmt.Errorf("version must be > 0")
	}
	if strings.TrimSpace(p.Source) == "" {
		return fmt.Errorf("source is required")
	}
	return nil
}
