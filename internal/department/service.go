package department

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"workplace/pkg/cache"
)

const (
	keyDeptList = "dept:list"
	ttlDept     = 30 * time.Minute
)

func deptKey(id string) string { return fmt.Sprintf("dept:id:%s", id) }

type Service struct {
	repo  *Repository
	cache *cache.Cache
}

func NewService(repo *Repository, c *cache.Cache) *Service {
	return &Service{repo: repo, cache: c}
}

// ErrValidation marks errors caused by bad input (shown to the client as 400).
var ErrValidation = errors.New("validation failed")

func invalid(msg string) error { return fmt.Errorf("%w: %s", ErrValidation, msg) }

// checkParent enforces one level of nesting: the parent must exist, be top-level
// itself, and not be the department being edited.
func (s *Service) checkParent(ctx context.Context, id string, parentID *string) error {
	if parentID == nil || *parentID == "" {
		return nil
	}
	if *parentID == id {
		return invalid("a department cannot be under itself")
	}
	parent, err := s.repo.FindByID(ctx, *parentID)
	if err != nil {
		return invalid("parent department not found")
	}
	if parent.ParentID != nil {
		return invalid("a sub-department cannot have its own sub-departments")
	}
	if id != "" {
		hasChildren, err := s.repo.HasChildren(ctx, id)
		if err != nil {
			return err
		}
		if hasChildren {
			return invalid("this department has sub-departments, so it cannot be placed under another")
		}
	}
	return nil
}

func duplicateName(err error) error {
	if err != nil && strings.Contains(err.Error(), "23505") {
		return invalid("a department with this name already exists")
	}
	return err
}

func (s *Service) Create(ctx context.Context, dto CreateDepartmentDTO) (*Department, error) {
	dto.Name = strings.TrimSpace(dto.Name)
	if dto.Name == "" {
		return nil, invalid("name is required")
	}
	if err := s.checkParent(ctx, "", dto.ParentID); err != nil {
		return nil, err
	}
	if dto.ParentID != nil && *dto.ParentID == "" {
		dto.ParentID = nil
	}
	if dto.HeadID != nil && *dto.HeadID == "" {
		dto.HeadID = nil
	}
	d := &Department{Name: dto.Name, ParentID: dto.ParentID, HeadID: dto.HeadID}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, duplicateName(err)
	}
	cache.Delete(ctx, s.cache, keyDeptList)
	return d, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Department, error) {
	if d, ok := cache.Get[Department](ctx, s.cache, deptKey(id)); ok {
		return d, nil
	}
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	cache.Set(ctx, s.cache, deptKey(id), d, ttlDept)
	return d, nil
}

func (s *Service) List(ctx context.Context) ([]*Department, error) {
	if list, ok := cache.Get[[]*Department](ctx, s.cache, keyDeptList); ok {
		return *list, nil
	}
	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	cache.Set(ctx, s.cache, keyDeptList, list, ttlDept)
	return list, nil
}

func (s *Service) Update(ctx context.Context, id string, dto UpdateDepartmentDTO) (*Department, error) {
	if dto.Name != nil {
		n := strings.TrimSpace(*dto.Name)
		if n == "" {
			return nil, invalid("name cannot be empty")
		}
		dto.Name = &n
	}
	if err := s.checkParent(ctx, id, dto.ParentID); err != nil {
		return nil, err
	}
	d, err := s.repo.Update(ctx, id, dto)
	if err != nil {
		return nil, duplicateName(err)
	}
	cache.Delete(ctx, s.cache, deptKey(id), keyDeptList)
	return d, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	cache.Delete(ctx, s.cache, deptKey(id), keyDeptList)
	return nil
}
