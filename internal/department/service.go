package department

import (
	"context"
	"errors"
	"fmt"
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

func (s *Service) Create(ctx context.Context, dto CreateDepartmentDTO) (*Department, error) {
	if dto.Name == "" {
		return nil, errors.New("name is required")
	}
	d := &Department{Name: dto.Name, HeadID: dto.HeadID}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
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
	d, err := s.repo.Update(ctx, id, dto)
	if err != nil {
		return nil, err
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
