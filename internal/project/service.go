package project

import (
	"context"
	"errors"
	"fmt"

	"workplace/internal/chat"
	"workplace/internal/notification"
	"workplace/internal/user"
)

type Service struct {
	repo        *Repository
	historyRepo *HistoryRepository
	budgetRepo  *BudgetRepository
	fileRepo    *FileRepository
	chatSvc     *chat.Service
	notifSvc    *notification.Service
	userRepo    *user.Repository
}

func NewService(repo *Repository, historyRepo *HistoryRepository, budgetRepo *BudgetRepository, fileRepo *FileRepository, chatSvc *chat.Service, notifSvc *notification.Service, userRepo *user.Repository) *Service {
	return &Service{repo: repo, historyRepo: historyRepo, budgetRepo: budgetRepo, fileRepo: fileRepo, chatSvc: chatSvc, notifSvc: notifSvc, userRepo: userRepo}
}

func (s *Service) Create(ctx context.Context, dto CreateProjectDTO, createdBy string) (*Project, error) {
	if dto.Title == "" {
		return nil, errors.New("title is required")
	}

	status := dto.Status
	if status == "" {
		status = StatusPending
	}

	p := &Project{
		Title:        dto.Title,
		Description:  dto.Description,
		Status:       status,
		StartDate:    dto.StartDate,
		EndDate:      dto.EndDate,
		DepartmentID: dto.DepartmentID,
		CreatedBy:    createdBy,
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}

	go s.chatSvc.CreateProjectRoom(context.Background(), p.ID, p.Title, createdBy)

	// Notify department members about the new project
	if p.DepartmentID != nil {
		go func(deptID, projectID, title string) {
			members, err := s.userRepo.ListByDepartment(context.Background(), deptID)
			if err != nil {
				return
			}
			refID := projectID
			var inputs []notification.NotifyInput
			for _, u := range members {
				if u.ID == createdBy {
					continue
				}
				inputs = append(inputs, notification.NotifyInput{
					UserID: u.ID,
					Type:   notification.TypeProjectCreated,
					Title:  "New Project: " + title,
					Body:   fmt.Sprintf("A new project \"%s\" has been created in your department.", title),
					RefID:  &refID,
				})
			}
			s.notifSvc.NotifyMany(context.Background(), inputs)
		}(*p.DepartmentID, p.ID, p.Title)
	}

	return p, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Project, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, deptID string) ([]*Project, error) {
	return s.repo.List(ctx, deptID)
}

func (s *Service) Update(ctx context.Context, id, changedBy string, dto UpdateProjectDTO) (*Project, error) {
	old, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	p, err := s.repo.Update(ctx, id, dto)
	if err != nil {
		return nil, err
	}
	if dto.Status != nil && *dto.Status != old.Status {
		from := string(old.Status)
		s.historyRepo.Log(ctx, &HistoryEntry{ProjectID: id, ChangedBy: changedBy, Field: "status", FromValue: &from, ToValue: string(*dto.Status)})

		go func(newStatus Status) {
			assignees, err := s.repo.GetAssignees(context.Background(), id)
			if err != nil {
				return
			}
			refID := id
			var inputs []notification.NotifyInput
			for _, uid := range assignees {
				if uid == changedBy {
					continue
				}
				inputs = append(inputs, notification.NotifyInput{
					UserID: uid,
					Type:   notification.TypeProjectStatus,
					Title:  "Project Status Changed — " + p.Title,
					Body:   fmt.Sprintf("Project \"%s\" status changed to %s.", p.Title, newStatus),
					RefID:  &refID,
				})
			}
			s.notifSvc.NotifyMany(context.Background(), inputs)
		}(*dto.Status)
	}
	return p, nil
}

func (s *Service) UpdateProgress(ctx context.Context, id, changedBy string, dto UpdateProgressDTO) error {
	if dto.Progress < 0 || dto.Progress > 100 {
		return errors.New("progress must be between 0 and 100")
	}
	old, _ := s.repo.FindByID(ctx, id)
	if err := s.repo.UpdateProgress(ctx, id, dto.Progress); err != nil {
		return err
	}
	if old != nil {
		from := itoa(old.Progress)
		s.historyRepo.Log(ctx, &HistoryEntry{ProjectID: id, ChangedBy: changedBy, Field: "progress", FromValue: &from, ToValue: itoa(dto.Progress)})
	}

	// Notify all assignees of the progress update — in-app + email
	go func() {
		p, err := s.repo.FindByID(ctx, id)
		if err != nil {
			return
		}
		assignees, err := s.repo.GetAssignees(ctx, id)
		if err != nil {
			return
		}
		refID := id
		body := fmt.Sprintf(`Project "%s" progress has been updated to %s%%.`, p.Title, itoa(dto.Progress))
		var inputs []notification.NotifyInput
		for _, uid := range assignees {
			u, err := s.userRepo.FindByID(ctx, uid)
			if err != nil {
				continue
			}
			inputs = append(inputs, notification.NotifyInput{
				UserID:    u.ID,
				Email:     u.Email,
				Name:      u.Name,
				Type:      notification.TypeProjectProgress,
				Title:     "Project Progress Updated — " + p.Title,
				Body:      body,
				RefID:     &refID,
				SendEmail: true,
			})
		}
		s.notifSvc.NotifyMany(ctx, inputs)
	}()

	return nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) Assign(ctx context.Context, projectID, userID string) error {
	if err := s.repo.Assign(ctx, projectID, userID); err != nil {
		return err
	}

	go func() {
		u, err := s.userRepo.FindByID(ctx, userID)
		if err != nil {
			return
		}
		p, err := s.repo.FindByID(ctx, projectID)
		if err != nil {
			return
		}
		refID := projectID
		s.notifSvc.Notify(ctx, notification.NotifyInput{
			UserID:    u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Type:      notification.TypeProjectAssigned,
			Title:     "Assigned to Project: " + p.Title,
			Body:      "You have been assigned to the project: " + p.Title,
			RefID:     &refID,
			SendEmail: true,
		})
		s.chatSvc.SyncProjectMember(ctx, projectID, userID, true)
	}()

	return nil
}

func (s *Service) Unassign(ctx context.Context, projectID, userID string) error {
	if err := s.repo.Unassign(ctx, projectID, userID); err != nil {
		return err
	}

	go func() {
		u, err := s.userRepo.FindByID(ctx, userID)
		if err != nil {
			return
		}
		p, err := s.repo.FindByID(ctx, projectID)
		if err != nil {
			return
		}
		refID := projectID
		s.notifSvc.Notify(ctx, notification.NotifyInput{
			UserID:    u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Type:      notification.TypeProjectUnassigned,
			Title:     "Removed from Project: " + p.Title,
			Body:      "You have been removed from the project: " + p.Title,
			RefID:     &refID,
			SendEmail: true,
		})
		s.chatSvc.SyncProjectMember(ctx, projectID, userID, false)
	}()

	return nil
}

func (s *Service) GetAssignees(ctx context.Context, projectID string) ([]string, error) {
	return s.repo.GetAssignees(ctx, projectID)
}

func (s *Service) GetProjectRoom(ctx context.Context, projectID string) (*chat.Room, error) {
	return s.chatSvc.GetProjectRoom(ctx, projectID)
}

func (s *Service) GetHistory(ctx context.Context, projectID string) ([]*HistoryEntry, error) {
	return s.historyRepo.List(ctx, projectID)
}

func (s *Service) SetBudget(ctx context.Context, projectID, changedBy string, dto SetBudgetDTO) error {
	if err := s.budgetRepo.SetBudget(ctx, projectID, dto.Budget); err != nil {
		return err
	}
	go func() {
		p, err := s.repo.FindByID(context.Background(), projectID)
		if err != nil {
			return
		}
		assignees, err := s.repo.GetAssignees(context.Background(), projectID)
		if err != nil {
			return
		}
		refID := projectID
		var inputs []notification.NotifyInput
		for _, uid := range assignees {
			if uid == changedBy {
				continue
			}
			inputs = append(inputs, notification.NotifyInput{
				UserID: uid,
				Type:   notification.TypeBudgetUpdated,
				Title:  "Budget Updated — " + p.Title,
				Body:   fmt.Sprintf("The budget for project \"%s\" has been updated.", p.Title),
				RefID:  &refID,
			})
		}
		s.notifSvc.NotifyMany(context.Background(), inputs)
	}()
	return nil
}

func (s *Service) GetBudget(ctx context.Context, projectID string) (*BudgetSummary, error) {
	return s.budgetRepo.GetSummary(ctx, projectID)
}

func (s *Service) AddExpense(ctx context.Context, projectID string, dto AddExpenseDTO, loggedBy string) (*Expense, error) {
	if dto.Description == "" || dto.Amount <= 0 {
		return nil, errors.New("description and a positive amount are required")
	}
	date := dto.Date
	e := &Expense{ProjectID: projectID, Description: dto.Description, Amount: dto.Amount, Date: date, LoggedBy: loggedBy}
	return e, s.budgetRepo.AddExpense(ctx, e)
}

func (s *Service) DeleteExpense(ctx context.Context, id string) error {
	return s.budgetRepo.DeleteExpense(ctx, id)
}

func (s *Service) AddFile(ctx context.Context, projectID string, dto AddFileDTO, uploadedBy string) (*File, error) {
	if dto.Name == "" || dto.URL == "" {
		return nil, errors.New("name and url are required")
	}
	f := &File{ProjectID: projectID, Name: dto.Name, URL: dto.URL, Size: dto.Size, UploadedBy: uploadedBy}
	return f, s.fileRepo.Add(ctx, f)
}

func (s *Service) ListFiles(ctx context.Context, projectID string) ([]*File, error) {
	return s.fileRepo.List(ctx, projectID)
}

func (s *Service) DeleteFile(ctx context.Context, id string) error {
	return s.fileRepo.Delete(ctx, id)
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
