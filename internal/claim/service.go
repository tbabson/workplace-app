package claim

import (
	"context"
	"errors"
	"fmt"

	"workplace/internal/notification"
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

func (s *Service) List(ctx context.Context, role, userID string) ([]*Claim, error) {
	if role == "staff" || role == "procurement" {
		return s.repo.List(ctx, userID)
	}
	return s.repo.List(ctx, "")
}

func (s *Service) GetByID(ctx context.Context, id string) (*Claim, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) Approve(ctx context.Context, id, reviewerID string) (*Claim, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("claim not found")
	}
	if c.Status != StatusPending {
		return nil, errors.New("only pending claims can be approved")
	}
	updated, err := s.repo.UpdateStatus(ctx, id, StatusApproved, reviewerID)
	if err != nil {
		return nil, err
	}

	go s.notifSvc.Notify(context.Background(), notification.NotifyInput{
		UserID: c.UserID,
		Type:   notification.TypeClaimApproved,
		Title:  "Expense Claim Approved",
		Body:   fmt.Sprintf("Your claim \"%s\" (%.2f) has been approved.", c.Title, c.Amount),
		RefID:  &c.ID,
	})
	return updated, nil
}

func (s *Service) Reject(ctx context.Context, id, reviewerID string) (*Claim, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("claim not found")
	}
	if c.Status != StatusPending {
		return nil, errors.New("only pending claims can be rejected")
	}
	updated, err := s.repo.UpdateStatus(ctx, id, StatusRejected, reviewerID)
	if err != nil {
		return nil, err
	}

	go s.notifSvc.Notify(context.Background(), notification.NotifyInput{
		UserID: c.UserID,
		Type:   notification.TypeClaimRejected,
		Title:  "Expense Claim Rejected",
		Body:   fmt.Sprintf("Your claim \"%s\" (%.2f) has been rejected.", c.Title, c.Amount),
		RefID:  &c.ID,
	})
	return updated, nil
}

func (s *Service) MarkPaid(ctx context.Context, id, reviewerID string) (*Claim, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("claim not found")
	}
	if c.Status != StatusApproved {
		return nil, errors.New("only approved claims can be marked as paid")
	}
	updated, err := s.repo.MarkPaid(ctx, id, reviewerID)
	if err != nil {
		return nil, err
	}

	go s.notifSvc.Notify(context.Background(), notification.NotifyInput{
		UserID: c.UserID,
		Type:   notification.TypeClaimPaid,
		Title:  "Expense Claim Paid",
		Body:   fmt.Sprintf("Your claim \"%s\" (%.2f) has been paid.", c.Title, c.Amount),
		RefID:  &c.ID,
	})
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
