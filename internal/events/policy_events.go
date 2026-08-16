package events

import (
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/google/uuid"
)

const (
	EventCreated   = "created"
	EventSubmitted = "submitted"
	EventActivated = "activated"
	EventRejected  = "rejected"
	EventAmended   = "amended"
	EventCancelled = "cancelled"
	EventExpired   = "expired"

	source = "ktayl-policy-service"
)

// PolicyEvent is the internal representation of a policy lifecycle event.
type PolicyEvent struct {
	EventType string
	PolicyID  string
	Policy    *domain.Policy
	Actor     string
	Reason    string
	OccurredAt time.Time
}

// cloudEvent is the CloudEvents 1.0 wire format (AC-2).
type cloudEvent struct {
	SpecVersion     string          `json:"specversion"`
	ID              string          `json:"id"`
	Source          string          `json:"source"`
	Type            string          `json:"type"`
	DataContentType string          `json:"datacontenttype"`
	Time            string          `json:"time"`
	Data            policyEventData `json:"data"`
}

type policyEventData struct {
	PolicyID      string `json:"policy_id"`
	PolicyNumber  string `json:"policy_number"`
	HolderName    string `json:"holder_name"`
	ProductCode   string `json:"product_code"`
	Status        string `json:"status"`
	Actor         string `json:"actor,omitempty"`
	Reason        string `json:"reason,omitempty"`
	EffectiveDate string `json:"effective_date"`
	ExpiryDate    string `json:"expiry_date"`
}

func (e *PolicyEvent) toCloudEvent() *cloudEvent {
	p := e.Policy
	return &cloudEvent{
		SpecVersion:     "1.0",
		ID:              uuid.NewString(),
		Source:          source,
		Type:            "com.ktayl.policy." + e.EventType,
		DataContentType: "application/json",
		Time:            e.OccurredAt.UTC().Format(time.RFC3339),
		Data: policyEventData{
			PolicyID:      p.ID.String(),
			PolicyNumber:  p.PolicyNumber,
			HolderName:    p.HolderName,
			ProductCode:   p.ProductCode,
			Status:        string(p.Status),
			Actor:         e.Actor,
			Reason:        e.Reason,
			EffectiveDate: p.EffectiveDate.UTC().Format(time.RFC3339),
			ExpiryDate:    p.ExpiryDate.UTC().Format(time.RFC3339),
		},
	}
}

// BuildTransitionEvent creates a PolicyEvent for a state machine transition.
func BuildTransitionEvent(eventType string, p *domain.Policy, actor, reason string) *PolicyEvent {
	return &PolicyEvent{
		EventType:  eventType,
		PolicyID:   p.ID.String(),
		Policy:     p,
		Actor:      actor,
		Reason:     reason,
		OccurredAt: p.UpdatedAt,
	}
}

// BuildCreatedEvent creates a PolicyEvent for a newly created policy.
func BuildCreatedEvent(p *domain.Policy, actor string) *PolicyEvent {
	return &PolicyEvent{
		EventType:  EventCreated,
		PolicyID:   p.ID.String(),
		Policy:     p,
		Actor:      actor,
		OccurredAt: p.CreatedAt,
	}
}

// eventTypeForDomainEvent maps a domain.Event to the CloudEvents event type string.
func EventTypeForTransition(to domain.PolicyStatus) string {
	switch to {
	case domain.StatusSubmitted:
		return EventSubmitted
	case domain.StatusActive:
		return EventActivated
	case domain.StatusRejected:
		return EventRejected
	case domain.StatusAmended:
		return EventAmended
	case domain.StatusCancelled:
		return EventCancelled
	case domain.StatusExpired:
		return EventExpired
	default:
		return string(to)
	}
}
