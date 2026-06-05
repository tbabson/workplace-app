package asset

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

func (s *Service) Create(ctx context.Context, dto CreateAssetDTO) (*Asset, error) {
	if dto.Name == "" || dto.Category == "" {
		return nil, errors.New("name and category are required")
	}
	a := &Asset{Name: dto.Name, Category: dto.Category, SerialNumber: dto.SerialNumber, Description: dto.Description}
	return a, s.repo.Create(ctx, a)
}

func (s *Service) List(ctx context.Context, role, userID string) ([]*Asset, error) {
	if role == "staff" || role == "procurement" {
		return s.repo.List(ctx, userID)
	}
	return s.repo.List(ctx, "")
}

func (s *Service) GetByID(ctx context.Context, id string) (*Asset, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id string, dto UpdateAssetDTO) (*Asset, error) {
	if dto.Status != nil {
		st := Status(*dto.Status)
		if st != StatusAvailable && st != StatusAssigned && st != StatusMaintenance && st != StatusRetired {
			return nil, errors.New("invalid status")
		}
	}
	return s.repo.Update(ctx, id, dto)
}

func (s *Service) Assign(ctx context.Context, id, userID string) (*Asset, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("asset not found")
	}
	if a.Status == StatusAssigned {
		return nil, errors.New("asset is already assigned")
	}
	if a.Status == StatusRetired {
		return nil, errors.New("cannot assign a retired asset")
	}
	updated, err := s.repo.Assign(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	go s.notifSvc.Notify(context.Background(), notification.NotifyInput{
		UserID: userID,
		Type:   notification.TypeAssetAssigned,
		Title:  "Asset Assigned to You",
		Body:   fmt.Sprintf("The asset \"%s\" (%s) has been assigned to you.", a.Name, a.Category),
		RefID:  &a.ID,
	})
	return updated, nil
}

func (s *Service) Unassign(ctx context.Context, id string) (*Asset, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("asset not found")
	}
	if a.Status != StatusAssigned {
		return nil, errors.New("asset is not currently assigned")
	}
	return s.repo.Unassign(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
