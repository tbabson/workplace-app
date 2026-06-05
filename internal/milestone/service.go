package milestone

import (
	"context"
	"errors"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, projectID string, dto CreateMilestoneDTO, createdBy string) (*Milestone, error) {
	if dto.Title == "" || dto.DueDate == "" {
		return nil, errors.New("title and due_date are required")
	}
	m := &Milestone{ProjectID: projectID, Title: dto.Title, DueDate: dto.DueDate, CreatedBy: createdBy}
	return m, s.repo.Create(ctx, m)
}

func (s *Service) ListByProject(ctx context.Context, projectID string) ([]*Milestone, error) {
	return s.repo.ListByProject(ctx, projectID)
}

func (s *Service) GetByID(ctx context.Context, id string) (*Milestone, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id string, dto UpdateMilestoneDTO) (*Milestone, error) {
	return s.repo.Update(ctx, id, dto)
}

func (s *Service) Achieve(ctx context.Context, id string) error {
	return s.repo.Achieve(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
