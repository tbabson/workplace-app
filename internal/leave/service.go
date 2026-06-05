package leave

import (
	"context"
	"errors"
	"fmt"
	"time"

	"workplace/internal/notification"
)

type Service struct {
	repo     *Repository
	notifSvc *notification.Service
}

func NewService(repo *Repository, notifSvc *notification.Service) *Service {
	return &Service{repo: repo, notifSvc: notifSvc}
}

func (s *Service) Create(ctx context.Context, dto CreateRequestDTO, userID string) (*Request, error) {
	if dto.Type == "" || dto.StartDate == "" || dto.EndDate == "" {
		return nil, errors.New("type, start_date, and end_date are required")
	}
	start, err := time.Parse("2006-01-02", dto.StartDate)
	if err != nil {
		return nil, errors.New("invalid start_date format, use YYYY-MM-DD")
	}
	end, err := time.Parse("2006-01-02", dto.EndDate)
	if err != nil {
		return nil, errors.New("invalid end_date format, use YYYY-MM-DD")
	}
	if end.Before(start) {
		return nil, errors.New("end_date must be after start_date")
	}
	days := int(end.Sub(start).Hours()/24) + 1
	req := &Request{UserID: userID, Type: dto.Type, StartDate: dto.StartDate, EndDate: dto.EndDate, Days: days, Reason: dto.Reason}
	return req, s.repo.Create(ctx, req)
}

func (s *Service) List(ctx context.Context, role, userID string) ([]*Request, error) {
	if role == "staff" || role == "procurement" {
		return s.repo.List(ctx, userID)
	}
	return s.repo.List(ctx, "")
}

func (s *Service) GetByID(ctx context.Context, id string) (*Request, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) Approve(ctx context.Context, id, reviewerID string, dto ReviewDTO) error {
	req, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return errors.New("leave request not found")
	}
	if req.Status != StatusPending {
		return errors.New("only pending requests can be approved")
	}
	note := &dto.Note
	if err := s.repo.Review(ctx, id, reviewerID, StatusApproved, note); err != nil {
		return err
	}
	year := time.Now().Year()
	_ = s.repo.DeductBalance(ctx, req.UserID, req.Type, year, req.Days)

	go s.notifSvc.Notify(context.Background(), notification.NotifyInput{
		UserID: req.UserID,
		Type:   notification.TypeLeaveApproved,
		Title:  "Leave Request Approved",
		Body:   fmt.Sprintf("Your %s leave request (%s – %s) has been approved.", req.Type, req.StartDate, req.EndDate),
		RefID:  &req.ID,
	})
	return nil
}

func (s *Service) Reject(ctx context.Context, id, reviewerID string, dto ReviewDTO) error {
	req, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return errors.New("leave request not found")
	}
	if req.Status != StatusPending {
		return errors.New("only pending requests can be rejected")
	}
	note := &dto.Note
	if err := s.repo.Review(ctx, id, reviewerID, StatusRejected, note); err != nil {
		return err
	}

	go s.notifSvc.Notify(context.Background(), notification.NotifyInput{
		UserID: req.UserID,
		Type:   notification.TypeLeaveRejected,
		Title:  "Leave Request Rejected",
		Body:   fmt.Sprintf("Your %s leave request (%s – %s) has been rejected.", req.Type, req.StartDate, req.EndDate),
		RefID:  &req.ID,
	})
	return nil
}

func (s *Service) Cancel(ctx context.Context, id, userID string) error {
	req, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return errors.New("leave request not found")
	}
	if req.UserID != userID {
		return errors.New("cannot cancel another user's request")
	}
	if req.Status != StatusPending {
		return errors.New("only pending requests can be cancelled")
	}
	return s.repo.Cancel(ctx, id)
}

func (s *Service) GetBalance(ctx context.Context, userID string, year int) ([]*Balance, error) {
	if year == 0 {
		year = time.Now().Year()
	}
	return s.repo.GetBalance(ctx, userID, year)
}

func (s *Service) SetBalance(ctx context.Context, dto SetBalanceDTO, userID string) error {
	if dto.Year == 0 {
		dto.Year = time.Now().Year()
	}
	return s.repo.SetBalance(ctx, &Balance{UserID: userID, Type: dto.Type, Year: dto.Year, Total: dto.Total})
}
