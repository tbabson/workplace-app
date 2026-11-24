package claim

import (
	"context"
	"errors"
	"fmt"

	"workplace/internal/notification"
	"workplace/pkg/utils"

	"github.com/jackc/pgx/v5"
)

var (
	ErrNotFound   = errors.New("claim not found")
	ErrForbidden  = errors.New("you are not allowed to act on this claim")
	ErrOwnClaim   = errors.New("you cannot review your own claim")
	ErrWrongState = errors.New("claim is not in a state that allows this action")
)

type Service struct {
	repo     *Repository
	notifSvc *notification.Service
}

func NewService(repo *Repository, notifSvc *notification.Service) *Service {
	return &Service{repo: repo, notifSvc: notifSvc}
}

func (s *Service) Create(ctx context.Context, dto CreateClaimDTO, userID string) (*Claim, error) {
	if dto.Title == "" || dto.Date == "" {
		return nil, errors.New("title and date are required")
	}
	if dto.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	if dto.ReceiptURL != nil && *dto.ReceiptURL != "" {
		if err := utils.ValidateHTTPURL(*dto.ReceiptURL); err != nil {
			return nil, fmt.Errorf("receipt_url: %w", err)
		}
	}
	c := &Claim{
		UserID:      userID,
		Title:       dto.Title,
		Amount:      dto.Amount,
		Date:        dto.Date,
		Description: dto.Description,
		ReceiptURL:  dto.ReceiptURL,
	}
	return c, s.repo.Create(ctx, c)
}

// List returns userID's claims when set, otherwise claims of users in deptID when
// set, otherwise all claims. Callers decide which scope a user may use.
func (s *Service) List(ctx context.Context, userID, deptID string) ([]*Claim, error) {
	return s.repo.List(ctx, userID, deptID)
}

// Get returns a claim. Access control is the caller's job.
func (s *Service) Get(ctx context.Context, id string) (*Claim, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *Service) transition(ctx context.Context, id, reviewerID string, from, to Status) (*Claim, *Claim, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, nil, ErrNotFound
	}
	if c.UserID == reviewerID {
		return nil, nil, ErrOwnClaim
	}
	var updated *Claim
	if to == StatusPaid {
		updated, err = s.repo.MarkPaid(ctx, id, reviewerID)
	} else {
		updated, err = s.repo.UpdateStatus(ctx, id, from, to, reviewerID)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrWrongState
	}
	return c, updated, err
}

func (s *Service) Approve(ctx context.Context, id, reviewerID string) (*Claim, error) {
	c, updated, err := s.transition(ctx, id, reviewerID, StatusPending, StatusApproved)
	if err != nil {
		return nil, err
	}
	s.notify(c, notification.TypeClaimApproved, "Expense Claim Approved", "approved")
	return updated, nil
}

func (s *Service) Reject(ctx context.Context, id, reviewerID string) (*Claim, error) {
	c, updated, err := s.transition(ctx, id, reviewerID, StatusPending, StatusRejected)
	if err != nil {
		return nil, err
	}
	s.notify(c, notification.TypeClaimRejected, "Expense Claim Rejected", "rejected")
	return updated, nil
}

func (s *Service) MarkPaid(ctx context.Context, id, reviewerID string) (*Claim, error) {
	c, updated, err := s.transition(ctx, id, reviewerID, StatusApproved, StatusPaid)
	if err != nil {
		return nil, err
	}
	s.notify(c, notification.TypeClaimPaid, "Expense Claim Paid", "paid")
	return updated, nil
}

func (s *Service) notify(c *Claim, typ notification.Type, title, verb string) {
	go s.notifSvc.Notify(context.Background(), notification.NotifyInput{
		UserID: c.UserID,
		Type:   typ,
		Title:  title,
		Body:   fmt.Sprintf("Your claim \"%s\" (%.2f) has been %s.", c.Title, c.Amount, verb),
		RefID:  &c.ID,
	})
}

// Delete lets a super_admin delete any claim, and an owner delete their own
// claim while it is still pending.
func (s *Service) Delete(ctx context.Context, id, callerID string, isAdmin bool) error {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if !isAdmin && (c.UserID != callerID || c.Status != StatusPending) {
		return ErrForbidden
	}
	return s.repo.Delete(ctx, id)
}
