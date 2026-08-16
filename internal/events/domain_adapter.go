package events

import (
	"context"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
)

// Ensure Publisher satisfies the domain.EventPublisher port.
var _ domain.EventPublisher = (*Publisher)(nil)

// PublishAsync builds the event envelope and calls Publish in a goroutine.
// context.WithoutCancel detaches cancellation so retries survive HTTP request
// completion while still propagating trace IDs and other context values.
func (p *Publisher) PublishAsync(ctx context.Context, eventType string, pol *domain.Policy, actor, reason string) {
	event := BuildTransitionEvent(eventType, pol, actor, reason)
	go p.Publish(context.WithoutCancel(ctx), event)
}
