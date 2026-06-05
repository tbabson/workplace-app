package notification

import (
	"context"
	"log"
)

type Service struct {
	repo   *Repository
	hub    *Hub
	mailer *Mailer
}

func NewService(repo *Repository, hub *Hub, mailer *Mailer) *Service {
	return &Service{repo: repo, hub: hub, mailer: mailer}
}

// Notify saves an in-app notification and pushes it to the user's WebSocket if connected.
func (s *Service) Notify(ctx context.Context, input NotifyInput) {
	n := &Notification{
		UserID: input.UserID,
		Type:   input.Type,
		Title:  input.Title,
		Body:   input.Body,
		RefID:  input.RefID,
	}

	if err := s.repo.Create(ctx, n); err != nil {
		log.Printf("notification create: %v", err)
		return
	}

	// Push real-time popup to connected client
	s.hub.Push(n)

	// Send email asynchronously — never blocks the caller
	if input.SendEmail && input.Email != "" {
		go func() {
			if err := s.mailer.Send(input.Email, input.Name, input.Title, input.Body); err != nil {
				log.Printf("notification email to %s: %v", input.Email, err)
			}
		}()
	}
}

// NotifyMany delivers the same notification to multiple users concurrently.
func (s *Service) NotifyMany(ctx context.Context, inputs []NotifyInput) {
	for _, inp := range inputs {
		go s.Notify(context.Background(), inp)
	}
}

func (s *Service) List(ctx context.Context, userID string) ([]*Notification, error) {
	return s.repo.ListForUser(ctx, userID)
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, id, userID string) error {
	return s.repo.MarkRead(ctx, id, userID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) ClearAll(ctx context.Context, userID string) error {
	return s.repo.ClearAll(ctx, userID)
}
