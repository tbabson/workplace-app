package projectcomment

import (
	"context"
	"errors"
	"fmt"

	"workplace/internal/notification"
	"workplace/internal/project"
)

type Service struct {
	repo        *Repository
	notifSvc    *notification.Service
	projectRepo *project.Repository
}

func NewService(repo *Repository, notifSvc *notification.Service, projectRepo *project.Repository) *Service {
	return &Service{repo: repo, notifSvc: notifSvc, projectRepo: projectRepo}
}

func (s *Service) Create(ctx context.Context, projectID string, dto CreateCommentDTO, authorID string) (*Comment, error) {
	if dto.Content == "" {
		return nil, errors.New("content is required")
	}
	c := &Comment{ProjectID: projectID, ParentID: dto.ParentID, AuthorID: authorID, Content: dto.Content}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}

	go func() {
		p, err := s.projectRepo.FindByID(context.Background(), projectID)
		if err != nil {
			return
		}
		assignees, err := s.projectRepo.GetAssignees(context.Background(), projectID)
		if err != nil {
			return
		}
		refID := projectID
		var inputs []notification.NotifyInput
		for _, uid := range assignees {
			if uid == authorID {
				continue // don't notify the person who commented
			}
			inputs = append(inputs, notification.NotifyInput{
				UserID: uid,
				Type:   notification.TypeProjectComment,
				Title:  "New Comment on " + p.Title,
				Body:   fmt.Sprintf("A new comment was added to project \"%s\".", p.Title),
				RefID:  &refID,
			})
		}
		s.notifSvc.NotifyMany(context.Background(), inputs)
	}()

	return c, nil
}

func (s *Service) ListByProject(ctx context.Context, projectID string) ([]*Comment, error) {
	return s.repo.ListByProject(ctx, projectID)
}

func (s *Service) Update(ctx context.Context, id, authorID string, dto UpdateCommentDTO) (*Comment, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("comment not found")
	}
	if c.AuthorID != authorID {
		return nil, errors.New("cannot edit another user's comment")
	}
	if dto.Content == "" {
		return nil, errors.New("content is required")
	}
	return s.repo.Update(ctx, id, dto.Content)
}

func (s *Service) Delete(ctx context.Context, id, callerID, callerRole string) error {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return errors.New("comment not found")
	}
	if c.AuthorID != callerID && callerRole != "super_admin" {
		return errors.New("cannot delete another user's comment")
	}
	return s.repo.Delete(ctx, id)
}
