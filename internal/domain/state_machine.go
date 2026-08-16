package domain

import "errors"

// Event is a business action that triggers a status transition (CdCF §6.1 BF-POL-03).
type Event string

const (
	EventSubmit   Event = "submit"
	EventActivate Event = "activate"
	EventReject   Event = "reject"
	EventAmend    Event = "amend"
	EventConfirm  Event = "confirm"
	EventCancel   Event = "cancel"
	EventExpire   Event = "expire"
)

// ReasonCode is the controlled vocabulary for audit log reason entries (DORA Art.9).
type ReasonCode string

const (
	ReasonUnderwriterApproved ReasonCode = "UNDERWRITER_APPROVED"
	ReasonUnderwriterRejected ReasonCode = "UNDERWRITER_REJECTED"
	ReasonCustomerRequest     ReasonCode = "CUSTOMER_REQUEST"
	ReasonNonPayment          ReasonCode = "NON_PAYMENT"
	ReasonRegulatory          ReasonCode = "REGULATORY"
)

// ErrInvalidTransition is returned when the requested event is not allowed from the current status.
var ErrInvalidTransition = errors.New("invalid state transition")

// transitionTable encodes the full lifecycle (CdCF §6.1 BF-POL-03).
var transitionTable = map[PolicyStatus]map[Event]PolicyStatus{
	StatusDraft: {
		EventSubmit: StatusSubmitted,
	},
	StatusSubmitted: {
		EventActivate: StatusActive,
		EventReject:   StatusRejected,
	},
	StatusActive: {
		EventAmend:  StatusAmended,
		EventCancel: StatusCancelled,
		EventExpire: StatusExpired,
	},
	StatusAmended: {
		EventConfirm: StatusActive,
	},
}

// Transition is a pure function: no side effects, no DB access.
// Returns the next status or ErrInvalidTransition.
func Transition(current PolicyStatus, event Event) (PolicyStatus, error) {
	events, ok := transitionTable[current]
	if !ok {
		return "", ErrInvalidTransition
	}
	next, ok := events[event]
	if !ok {
		return "", ErrInvalidTransition
	}
	return next, nil
}
