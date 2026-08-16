package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPolicyNotFound = errors.New("policy not found")
	ErrCannotCancel   = errors.New("only draft policies can be cancelled via this endpoint")
	ErrReasonRequired = errors.New("reason is required for this transition")
)

// EventPublisher is the port for async event publishing (implemented by events.Publisher).
// The interface lives in domain to avoid a circular import (events imports domain).
type EventPublisher interface {
	// PublishAsync sends the event in a goroutine after DB commit (AC-4).
	PublishAsync(ctx context.Context, eventType string, p *Policy, actor, reason string)
	// IsConnected reports whether the underlying transport is healthy.
	IsConnected() bool
}

type PolicyService struct {
	repo      PolicyRepository
	auditRepo AuditLogRepository
	db        *pgxpool.Pool
	pub       EventPublisher // optional; nil = no events
}

func NewPolicyService(repo PolicyRepository, auditRepo AuditLogRepository, db *pgxpool.Pool) *PolicyService {
	return &PolicyService{repo: repo, auditRepo: auditRepo, db: db}
}

// SetPublisher attaches an event publisher to this service. Called from main.go after construction.
func (s *PolicyService) SetPublisher(pub EventPublisher) { s.pub = pub }

// NATSConnected returns true when a publisher is wired and connected.
func (s *PolicyService) NATSConnected() bool {
	return s.pub != nil && s.pub.IsConnected()
}

func (s *PolicyService) Create(ctx context.Context, p *Policy, actorID string) error {
	if err := p.Validate(); err != nil {
		return err
	}
	p.ID = uuid.New()
	p.Status = StatusDraft
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	if err := s.repo.Create(ctx, p); err != nil {
		return err
	}
	// AC-4: publish only after successful DB write
	if s.pub != nil {
		s.pub.PublishAsync(ctx, "created", p, actorID, "")
	}
	return nil
}

func (s *PolicyService) GetByID(ctx context.Context, id uuid.UUID) (*Policy, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrPolicyNotFound
		}
		return nil, err
	}
	return p, nil
}

func (s *PolicyService) List(ctx context.Context, params ListParams) ([]*Policy, error) {
	return s.repo.List(ctx, params)
}

func (s *PolicyService) Update(ctx context.Context, id uuid.UUID, upd *Policy) (*Policy, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrPolicyNotFound
		}
		return nil, err
	}
	p.HolderName = upd.HolderName
	p.ProductCode = upd.ProductCode
	p.EffectiveDate = upd.EffectiveDate
	p.ExpiryDate = upd.ExpiryDate
	p.UpdatedAt = time.Now().UTC()
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Cancel soft-cancels a DRAFT policy by moving it to terminated.
// State machine transitions (submit/activate) are handled via Transition methods.
func (s *PolicyService) Cancel(ctx context.Context, id uuid.UUID) error {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrPolicyNotFound
		}
		return err
	}
	if p.Status != StatusDraft {
		return ErrCannotCancel
	}
	p.Status = StatusTerminated
	p.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, p)
}

// Submit transitions a DRAFT policy to SUBMITTED.
func (s *PolicyService) Submit(ctx context.Context, id uuid.UUID, actorID string) (*Policy, error) {
	return s.applyTransition(ctx, id, EventSubmit, actorID, "")
}

// Activate transitions a SUBMITTED policy to ACTIVE (underwriter approval).
func (s *PolicyService) Activate(ctx context.Context, id uuid.UUID, actorID string) (*Policy, error) {
	return s.applyTransition(ctx, id, EventActivate, actorID, string(ReasonUnderwriterApproved))
}

// Reject transitions a SUBMITTED policy to REJECTED (underwriter rejection).
func (s *PolicyService) Reject(ctx context.Context, id uuid.UUID, actorID, reason string) (*Policy, error) {
	if reason == "" {
		reason = string(ReasonUnderwriterRejected)
	}
	return s.applyTransition(ctx, id, EventReject, actorID, reason)
}

// CancelActive transitions an ACTIVE policy to CANCELLED; reason is mandatory.
func (s *PolicyService) CancelActive(ctx context.Context, id uuid.UUID, actorID, reason string) (*Policy, error) {
	if reason == "" {
		return nil, ErrReasonRequired
	}
	return s.applyTransition(ctx, id, EventCancel, actorID, reason)
}

// Expire transitions an ACTIVE policy to EXPIRED (batch use).
func (s *PolicyService) Expire(ctx context.Context, id uuid.UUID, actorID string) (*Policy, error) {
	return s.applyTransition(ctx, id, EventExpire, actorID, string(ReasonRegulatory))
}

// ListHistory returns the ordered audit trail for a policy.
func (s *PolicyService) ListHistory(ctx context.Context, id uuid.UUID) ([]*AuditLog, error) {
	// Verify the policy exists first to give a clean 404.
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrPolicyNotFound
		}
		return nil, err
	}
	return s.auditRepo.ListByPolicy(ctx, id)
}

// applyTransition runs Transition, then in a single DB tx: UPDATE policies + INSERT audit_log.
func (s *PolicyService) applyTransition(ctx context.Context, id uuid.UUID, event Event, actorID, reason string) (*Policy, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrPolicyNotFound
		}
		return nil, err
	}

	nextStatus, err := Transition(p.Status, event)
	if err != nil {
		return nil, fmt.Errorf("%w: %s → %s", ErrInvalidTransition, p.Status, event)
	}

	from := p.Status
	p.Status = nextStatus
	p.UpdatedAt = time.Now().UTC()

	if s.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx,
		`UPDATE policies SET status = $1, updated_at = $2 WHERE id = $3`,
		string(nextStatus), p.UpdatedAt, id,
	)
	if err != nil {
		return nil, err
	}

	entry := &AuditLog{
		PolicyID:   id,
		FromStatus: from,
		ToStatus:   nextStatus,
		ActorID:    actorID,
		Reason:     reason,
		OccurredAt: p.UpdatedAt,
	}
	if err := s.auditRepo.Insert(ctx, tx, entry); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	// AC-4: publish only after commit succeeds (no phantom events on rollback)
	if s.pub != nil {
		eventType := string(nextStatus) // fallback
		switch nextStatus {
		case StatusSubmitted:
			eventType = "submitted"
		case StatusActive:
			eventType = "activated"
		case StatusRejected:
			eventType = "rejected"
		case StatusAmended:
			eventType = "amended"
		case StatusCancelled:
			eventType = "cancelled"
		case StatusExpired:
			eventType = "expired"
		}
		s.pub.PublishAsync(ctx, eventType, p, actorID, reason)
	}
	return p, nil
}
