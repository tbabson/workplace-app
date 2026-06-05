package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"workplace/pkg/cache"
	"workplace/pkg/utils"
)

const (
	ttlUser = 10 * time.Minute
)

func userKey(id string) string    { return fmt.Sprintf("user:id:%s", id) }
func emailKey(email string) string { return fmt.Sprintf("user:email:%s", email) }

type Service struct {
	repo  *Repository
	cache *cache.Cache
}

func NewService(repo *Repository, c *cache.Cache) *Service {
	return &Service{repo: repo, cache: c}
}

func (s *Service) Create(ctx context.Context, dto CreateUserDTO) (*User, error) {
	if dto.Name == "" || dto.Email == "" || dto.Password == "" {
		return nil, errors.New("name, email and password are required")
	}

	hash, err := utils.HashPassword(dto.Password)
	if err != nil {
		return nil, err
	}

	role := dto.Role
	if role == "" {
		role = RoleStaff
	}

	u := &User{
		Name:           dto.Name,
		Email:          dto.Email,
		PasswordHash:   hash,
		Role:           role,
		Position:       dto.Position,
		DepartmentID:   dto.DepartmentID,
		DepartmentName: dto.DepartmentName,
		IsActive:       true,
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	if u, ok := cache.Get[User](ctx, s.cache, userKey(id)); ok {
		return u, nil
	}
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	cache.Set(ctx, s.cache, userKey(id), u, ttlUser)
	return u, nil
}

func (s *Service) GetByEmail(ctx context.Context, email string) (*User, error) {
	if u, ok := cache.Get[User](ctx, s.cache, emailKey(email)); ok {
		return u, nil
	}
	u, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	cache.Set(ctx, s.cache, emailKey(email), u, ttlUser)
	return u, nil
}

func (s *Service) List(ctx context.Context) ([]*User, error) {
	return s.repo.List(ctx)
}

func (s *Service) ListByDepartment(ctx context.Context, deptID string) ([]*User, error) {
	return s.repo.ListByDepartment(ctx, deptID)
}

func (s *Service) Update(ctx context.Context, id string, dto UpdateUserDTO) (*User, error) {
	u, err := s.repo.Update(ctx, id, dto)
	if err != nil {
		return nil, err
	}
	// Invalidate both keys — email may have been indirectly stale too
	cache.Delete(ctx, s.cache, userKey(id), emailKey(u.Email))
	return u, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	u, _ := s.repo.FindByID(ctx, id)
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	keys := []string{userKey(id)}
	if u != nil {
		keys = append(keys, emailKey(u.Email))
	}
	cache.Delete(ctx, s.cache, keys...)
	return nil
}
