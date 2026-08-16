package domain_test

import (
	"errors"
	"testing"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
)

func TestTransition_ValidTransitions(t *testing.T) {
	cases := []struct {
		from  domain.PolicyStatus
		event domain.Event
		to    domain.PolicyStatus
	}{
		{domain.StatusDraft, domain.EventSubmit, domain.StatusSubmitted},
		{domain.StatusSubmitted, domain.EventActivate, domain.StatusActive},
		{domain.StatusSubmitted, domain.EventReject, domain.StatusRejected},
		{domain.StatusActive, domain.EventAmend, domain.StatusAmended},
		{domain.StatusActive, domain.EventCancel, domain.StatusCancelled},
		{domain.StatusActive, domain.EventExpire, domain.StatusExpired},
		{domain.StatusAmended, domain.EventConfirm, domain.StatusActive},
	}
	for _, tc := range cases {
		t.Run(string(tc.from)+"→"+string(tc.event), func(t *testing.T) {
			got, err := domain.Transition(tc.from, tc.event)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.to {
				t.Errorf("want %s, got %s", tc.to, got)
			}
		})
	}
}

func TestTransition_InvalidTransitions(t *testing.T) {
	cases := []struct {
		from  domain.PolicyStatus
		event domain.Event
	}{
		// Draft cannot be directly activated or cancelled
		{domain.StatusDraft, domain.EventActivate},
		{domain.StatusDraft, domain.EventCancel},
		{domain.StatusDraft, domain.EventExpire},
		// Submitted cannot be amended or expired
		{domain.StatusSubmitted, domain.EventAmend},
		{domain.StatusSubmitted, domain.EventExpire},
		{domain.StatusSubmitted, domain.EventCancel},
		// Terminal states cannot transition
		{domain.StatusRejected, domain.EventSubmit},
		{domain.StatusCancelled, domain.EventActivate},
		{domain.StatusExpired, domain.EventActivate},
		// Amended cannot be submitted or rejected
		{domain.StatusAmended, domain.EventSubmit},
		{domain.StatusAmended, domain.EventReject},
	}
	for _, tc := range cases {
		t.Run(string(tc.from)+"→"+string(tc.event), func(t *testing.T) {
			_, err := domain.Transition(tc.from, tc.event)
			if !errors.Is(err, domain.ErrInvalidTransition) {
				t.Fatalf("expected ErrInvalidTransition, got %v", err)
			}
		})
	}
}
