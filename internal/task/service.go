package task

import (
	"context"
	"errors"
	"fmt"

	"workplace/internal/chat"
	"workplace/internal/notification"
)

// ProjectMembership reports whether a user is assigned to a project.
// project.Repository implements it.
type ProjectMembership interface {
	IsAssigned(ctx context.Context, projectID, userID string) (bool, error)
}

type Service struct {
	repo     *Repository
	notifSvc *notification.Service
	chatSvc  *chat.Service
	projects ProjectMembership
}

func NewService(repo *Repository, notifSvc *notification.Service, chatSvc *chat.Service, projects ProjectMembership) *Service {
	return &Service{repo: repo, notifSvc: notifSvc, chatSvc: chatSvc, projects: projects}
}

// IsProjectMember reports whether userID is assigned to projectID.
func (s *Service) IsProjectMember(ctx context.Context, projectID, userID string) (bool, error) {
	return s.projects.IsAssigned(ctx, projectID, userID)
}

func (s *Service) Create(ctx context.Context, projectID string, dto CreateTaskDTO, createdBy string) (*Task, error) {
	if dto.Title == "" {
		return nil, errors.New("title is required")
	}
	t := &Task{
		ProjectID:   projectID,
		ParentID:    dto.ParentID,
		Title:       dto.Title,
		Description: dto.Description,
		AssignedTo:  dto.AssignedTo,
		DueDate:     dto.DueDate,
		CreatedBy:   createdBy,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	if t.AssignedTo != nil && *t.AssignedTo != createdBy {
		assignee := *t.AssignedTo
		refID := t.ID
		go func() {
			_ = s.chatSvc.SyncProjectMember(context.Background(), t.ProjectID, assignee, true)
			s.notifSvc.Notify(context.Background(), notification.NotifyInput{
				UserID: assignee,
				Type:   notification.TypeTaskAssigned,
				Title:  "Task Assigned to You",
				Body:   fmt.Sprintf("You have been assigned the task: \"%s\"", t.Title),
				RefID:  &refID,
			})
		}()
	}
	return t, nil
}

func (s *Service) ListByProject(ctx context.Context, projectID string) ([]*Task, error) {
	return s.repo.ListByProject(ctx, projectID)
}

func (s *Service) GetByID(ctx context.Context, id string) (*Task, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id string, dto UpdateTaskDTO) (*Task, error) {
	old, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("task not found")
	}
	updated, err := s.repo.Update(ctx, id, dto)
	if err != nil {
		return nil, err
	}

	// Notify new assignee if assignment changed and add them to the project chat room
	if dto.AssignedTo != nil {
		newAssignee := *dto.AssignedTo
		oldAssignee := ""
		if old.AssignedTo != nil {
			oldAssignee = *old.AssignedTo
		}
		if newAssignee != oldAssignee {
			refID := id
			go func() {
				_ = s.chatSvc.SyncProjectMember(context.Background(), updated.ProjectID, newAssignee, true)
				s.notifSvc.Notify(context.Background(), notification.NotifyInput{
					UserID: newAssignee,
					Type:   notification.TypeTaskAssigned,
					Title:  "Task Assigned to You",
					Body:   fmt.Sprintf("You have been assigned the task: \"%s\"", updated.Title),
					RefID:  &refID,
				})
			}()
		}
	}

	// Notify task creator if status changed (and they are not the one updating)
	if dto.Status != nil && *dto.Status != old.Status {
		refID := id
		go s.notifSvc.Notify(context.Background(), notification.NotifyInput{
			UserID: old.CreatedBy,
			Type:   notification.TypeTaskUpdated,
			Title:  "Task Status Updated",
			Body:   fmt.Sprintf("Task \"%s\" status changed to %s.", updated.Title, *dto.Status),
			RefID:  &refID,
		})
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
