package events

import (
	"context"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
)

// Ensure Publisher satisfies the domain.EventPublisher port.
var _ domain.EventPublisher = (*Publisher)(nil)

// PublishAsync builds the event envelope and calls Publish in a goroutine.
// The goroutine inherits a background context so it is not cancelled if the
// HTTP request context is already done when retries kick in.
func (p *Publisher) PublishAsync(ctx context.Context, eventType string, pol *domain.Policy, actor, reason string) {
	event := BuildTransitionEvent(eventType, pol, actor, reason)
	go p.Publish(context.Background(), event)
}
