package leave

import (
	"context"
	"errors"
	"fmt"
	"time"

	"workplace/internal/notification"

	"github.com/jackc/pgx/v5"
)

var (
	ErrNotFound   = errors.New("leave request not found")
	ErrOwnRequest = errors.New("you cannot review your own leave request")
	ErrForbidden  = errors.New("cannot cancel another user's request")
	ErrNotPending = errors.New("only pending requests can be changed")
)

type Service struct {
	repo     *Repository
	notifSvc *notification.Service
}

func NewService(repo *Repository, notifSvc *notification.Service) *Service {
	return &Service{repo: repo, notifSvc: notifSvc}
}

func validType(t Type) bool {
	switch t {
	case TypeAnnual, TypeSick, TypeMaternity, TypePaternity, TypeUnpaid, TypeOther:
		return true
	}
	return false
}

// workingDays counts Monday–Friday days in [start, end].
func workingDays(start, end time.Time) int {
	n := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			n++
		}
	}
	return n
}

func (s *Service) Create(ctx context.Context, dto CreateRequestDTO, userID string) (*Request, error) {
	if dto.Type == "" || dto.StartDate == "" || dto.EndDate == "" {
		return nil, errors.New("type, start_date, and end_date are required")
	}
	if !validType(dto.Type) {
		return nil, errors.New("invalid leave type")
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
	days := workingDays(start, end)
	if days == 0 {
		return nil, errors.New("leave period contains no working days")
	}
	req := &Request{UserID: userID, Type: dto.Type, StartDate: dto.StartDate, EndDate: dto.EndDate, Days: days, Reason: dto.Reason}
	return req, s.repo.Create(ctx, req)
}

// List returns all requests when seeAll is true, otherwise only the caller's own.
func (s *Service) List(ctx context.Context, seeAll bool, userID string) ([]*Request, error) {
	if seeAll {
		return s.repo.List(ctx, "")
	}
	return s.repo.List(ctx, userID)
}

func (s *Service) Get(ctx context.Context, id, callerID string, seeAll bool) (*Request, error) {
	req, err := s.repo.FindByID(ctx, id)
	if err != nil || (!seeAll && req.UserID != callerID) {
		return nil, ErrNotFound
	}
	return req, nil
}

func (s *Service) Approve(ctx context.Context, id, reviewerID string, dto ReviewDTO) error {
	req, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if req.UserID == reviewerID {
		return ErrOwnRequest
	}
	// Deduct from the balance of the year the leave starts in, not the current year.
	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return err
	}
	note := &dto.Note
	if err := s.repo.Approve(ctx, req, reviewerID, note, start.Year()); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotPending
		}
		return err
	}

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
		return ErrNotFound
	}
	if req.UserID == reviewerID {
		return ErrOwnRequest
	}
	note := &dto.Note
	if err := s.repo.Review(ctx, id, reviewerID, StatusRejected, note); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotPending
		}
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
		return ErrNotFound
	}
	if req.UserID != userID {
		return ErrForbidden
	}
	if err := s.repo.Cancel(ctx, id, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotPending
		}
		return err
	}
	return nil
}

func (s *Service) GetBalance(ctx context.Context, userID string, year int) ([]*Balance, error) {
	if year == 0 {
		year = time.Now().Year()
	}
	return s.repo.GetBalance(ctx, userID, year)
}

func (s *Service) SetBalance(ctx context.Context, dto SetBalanceDTO, userID string) error {
	if !validType(dto.Type) {
		return errors.New("invalid leave type")
	}
	if dto.Total < 0 {
		return errors.New("total must not be negative")
	}
	if dto.Year == 0 {
		dto.Year = time.Now().Year()
	}
	return s.repo.SetBalance(ctx, &Balance{UserID: userID, Type: dto.Type, Year: dto.Year, Total: dto.Total})
}
