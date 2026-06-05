package review

import (
	"context"
	"errors"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, dto CreateReviewDTO, reviewerID string) (*Review, error) {
	if dto.UserID == "" || dto.Period == "" {
		return nil, errors.New("user_id and period are required")
	}
	if dto.Rating < 1 || dto.Rating > 5 {
		return nil, errors.New("rating must be between 1 and 5")
	}
	rv := &Review{UserID: dto.UserID, ReviewerID: reviewerID, Period: dto.Period, Rating: dto.Rating, Goals: dto.Goals, Comments: dto.Comments}
	return rv, s.repo.Create(ctx, rv)
}

func (s *Service) List(ctx context.Context, role, userID string) ([]*Review, error) {
	if role == "staff" {
		return s.repo.List(ctx, userID)
	}
	return s.repo.List(ctx, "")
}

func (s *Service) GetByID(ctx context.Context, id string) (*Review, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id string, dto UpdateReviewDTO) (*Review, error) {
	if dto.Rating != nil && (*dto.Rating < 1 || *dto.Rating > 5) {
		return nil, errors.New("rating must be between 1 and 5")
	}
	return s.repo.Update(ctx, id, dto)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
