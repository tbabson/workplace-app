package query

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

func (s *Service) Create(ctx context.Context, dto CreateQueryDTO, issuerID string) (*Query, error) {
	if dto.Title == "" || dto.Content == "" || dto.RecipientID == "" {
		return nil, errors.New("title, content and recipient_id are required")
	}

	q := &Query{
		Title:       dto.Title,
		Content:     dto.Content,
		IssuerID:    issuerID,
		RecipientID: dto.RecipientID,
	}

	if err := s.repo.Create(ctx, q); err != nil {
		return nil, err
	}

	// Notify recipient in-app + email
	go func() {
		u, err := s.userRepo.FindByID(ctx, dto.RecipientID)
		if err != nil {
			return
		}
		refID := q.ID
		s.notifSvc.Notify(ctx, notification.NotifyInput{
			UserID:    u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Type:      notification.TypeQueryIssued,
			Title:     "Query Issued: " + q.Title,
			Body:      q.Content,
			RefID:     &refID,
			SendEmail: true,
		})
	}()

	return q, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Query, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, role, userID string) ([]*Query, error) {
	if role == "super_admin" {
		return s.repo.ListAll(ctx)
	}
	return s.repo.ListForUser(ctx, userID)
}

func (s *Service) Respond(ctx context.Context, id, responderID string, dto RespondDTO) (*Query, error) {
	q, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("query not found")
	}

	if q.RecipientID != responderID {
		return nil, errors.New("only the recipient can respond to this query")
	}

	if dto.Status == "" {
		dto.Status = StatusAcknowledged
	}

	updated, err := s.repo.Respond(ctx, id, dto)
	if err != nil {
		return nil, err
	}

	// Notify the issuer that their query was responded to
	go func() {
		u, err := s.userRepo.FindByID(ctx, q.IssuerID)
		if err != nil {
			return
		}
		refID := id
		s.notifSvc.Notify(ctx, notification.NotifyInput{
			UserID:    u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Type:      notification.TypeQueryResponded,
			Title:     "Query Responded: " + q.Title,
			Body:      dto.Response,
			RefID:     &refID,
			SendEmail: true,
		})
	}()

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
