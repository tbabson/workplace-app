package memo

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

func (s *Service) Create(ctx context.Context, dto CreateMemoDTO, senderID string) (*Memo, error) {
	if dto.Title == "" || dto.Content == "" {
		return nil, errors.New("title and content are required")
	}

	m := &Memo{
		Title:        dto.Title,
		Content:      dto.Content,
		SenderID:     senderID,
		RecipientID:  dto.RecipientID,
		DepartmentID: dto.DepartmentID,
	}

	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}

	go s.sendMemoNotifications(context.Background(), m, senderID)

	return m, nil
}

func (s *Service) sendMemoNotifications(ctx context.Context, m *Memo, senderID string) {
	refID := m.ID
	title := "New Memo: " + m.Title

	buildInput := func(u *user.User) notification.NotifyInput {
		return notification.NotifyInput{
			UserID:    u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Type:      notification.TypeMemoReceived,
			Title:     title,
			Body:      m.Content,
			RefID:     &refID,
			SendEmail: true,
		}
	}

	switch {
	case m.RecipientID != nil:
		// Individual recipient
		u, err := s.userRepo.FindByID(ctx, *m.RecipientID)
		if err != nil {
			return
		}
		s.notifSvc.Notify(ctx, buildInput(u))

	case m.DepartmentID != nil:
		// All users in the department (except the sender)
		users, err := s.userRepo.ListByDepartment(ctx, *m.DepartmentID)
		if err != nil {
			return
		}
		var inputs []notification.NotifyInput
		for _, u := range users {
			if u.ID == senderID {
				continue
			}
			inputs = append(inputs, buildInput(u))
		}
		s.notifSvc.NotifyMany(ctx, inputs)

	default:
		// Company-wide broadcast — notify everyone except the sender
		users, err := s.userRepo.List(ctx)
		if err != nil {
			return
		}
		var inputs []notification.NotifyInput
		for _, u := range users {
			if u.ID == senderID {
				continue
			}
			inputs = append(inputs, buildInput(u))
		}
		s.notifSvc.NotifyMany(ctx, inputs)
	}
}

func (s *Service) GetByID(ctx context.Context, id string) (*Memo, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, role, userID, deptID string) ([]*Memo, error) {
	var memos []*Memo
	var err error
	if role == "super_admin" {
		memos, err = s.repo.ListAll(ctx)
	} else {
		memos, err = s.repo.ListForUser(ctx, userID, deptID)
	}
	if err != nil {
		return nil, err
	}

	// Enrich each memo with per-user acknowledgement state and counts
	for _, m := range memos {
		m.AcknowledgedAt, _ = s.repo.IsAcknowledged(ctx, m.ID, userID)

		acksCount, _ := s.repo.CountAcknowledgements(ctx, m.ID)
		totalCount, _ := s.repo.CountRecipients(ctx, m)
		m.AcknowledgedCount = acksCount
		m.PendingCount = totalCount - acksCount
		if m.PendingCount < 0 {
			m.PendingCount = 0
		}
	}

	return memos, nil
}

func (s *Service) Acknowledge(ctx context.Context, memoID, userID string) error {
	return s.repo.Acknowledge(ctx, memoID, userID)
}

func (s *Service) GetAcknowledgements(ctx context.Context, memoID string) ([]Acknowledgement, error) {
	return s.repo.ListAcknowledgements(ctx, memoID)
}

func (s *Service) MarkRead(ctx context.Context, id string) error {
	return s.repo.MarkRead(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
