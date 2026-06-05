package overtime

import (
	"context"
	"errors"

	"workplace/internal/notification"
	"workplace/internal/user"
)

type Service struct {
	repo     *Repository
	notifSvc *notification.Service
	userRepo *user.Repository
}

func NewService(repo *Repository, notifSvc *notification.Service, userRepo *user.Repository) *Service {
	return &Service{repo: repo, notifSvc: notifSvc, userRepo: userRepo}
}

func (s *Service) Create(ctx context.Context, dto CreateOvertimeDTO, createdBy string) (*Overtime, error) {
	if dto.UserID == "" {
		return nil, errors.New("user_id is required")
	}

	var hours *float64
	if dto.EndTime != nil {
		h := dto.EndTime.Sub(dto.StartTime).Hours()
		if h < 0 {
			return nil, errors.New("end_time must be after start_time")
		}
		hours = &h
	}

	o := &Overtime{
		UserID:     dto.UserID,
		ProjectID:  dto.ProjectID,
		StartTime:  dto.StartTime,
		EndTime:    dto.EndTime,
		Hours:      hours,
		HourlyRate: dto.HourlyRate,
		Notes:      dto.Notes,
		CreatedBy:  createdBy,
	}

	result, err := s.repo.Create(ctx, o)
	if err != nil {
		return nil, err
	}

	// Notify staff member their overtime has been logged
	go func() {
		bgCtx := context.Background()
		u, err := s.userRepo.FindByID(bgCtx, dto.UserID)
		if err != nil {
			return
		}
		refID := result.ID
		s.notifSvc.Notify(bgCtx, notification.NotifyInput{
			UserID:    u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Type:      notification.TypeOvertimeCreated,
			Title:     "Overtime Logged",
			Body:      "An overtime record has been created for you. Notes: " + dto.Notes,
			RefID:     &refID,
			SendEmail: true,
		})
	}()

	return result, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Overtime, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, userID string) ([]*Overtime, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) Update(ctx context.Context, id string, dto UpdateOvertimeDTO) (*Overtime, error) {
	return s.repo.Update(ctx, id, dto)
}

func (s *Service) MarkPaid(ctx context.Context, id string) error {
	o, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.MarkPaid(ctx, id); err != nil {
		return err
	}

	// Notify staff member their overtime has been paid
	go func() {
		bgCtx := context.Background()
		u, err := s.userRepo.FindByID(bgCtx, o.UserID)
		if err != nil {
			return
		}
		refID := id
		s.notifSvc.Notify(bgCtx, notification.NotifyInput{
			UserID:    u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Type:      notification.TypeOvertimePaid,
			Title:     "Overtime Payment Processed",
			Body:      "Your overtime payment has been marked as paid.",
			RefID:     &refID,
			SendEmail: true,
		})
	}()

	return nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
