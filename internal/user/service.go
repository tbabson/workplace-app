package user

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"strings"
	"time"

	"workplace/pkg/cache"
	"workplace/pkg/middleware"
	"workplace/pkg/utils"
)

const (
	ttlUser        = 10 * time.Minute
	minPasswordLen = 8
)

func userKey(id string) string { return fmt.Sprintf("user:id:%s", id) }

type Service struct {
	repo  *Repository
	cache *cache.Cache

	// onDepartmentChanged runs when a user's effective department changes. See
	// OnDepartmentChanged.
	onDepartmentChanged func(ctx context.Context, userID, fromDept, toDept string)
}

// OnDepartmentChanged registers a callback for a user's department changing,
// used to keep their project memberships in step with their department. It runs
// when a user is created in a department (from ""), moved between departments,
// removed from one (to ""), or reactivated in one (from ""). The project package
// can't be imported here without a cycle, so main wires it up.
func (s *Service) OnDepartmentChanged(fn func(ctx context.Context, userID, fromDept, toDept string)) {
	s.onDepartmentChanged = fn
}

// effectiveDept is the department a user counts as being in: none while inactive.
func effectiveDept(u *User) string {
	if u == nil || !u.IsActive || u.DepartmentID == nil {
		return ""
	}
	return *u.DepartmentID
}

func (s *Service) departmentChanged(ctx context.Context, before, after *User) {
	from, to := effectiveDept(before), effectiveDept(after)
	if s.onDepartmentChanged != nil && from != to {
		s.onDepartmentChanged(ctx, after.ID, from, to)
	}
}

func NewService(repo *Repository, c *cache.Cache) *Service {
	return &Service{repo: repo, cache: c}
}

func (r Role) Valid() bool {
	switch r {
	case RoleSuperAdmin, RoleDeptHead, RoleStaff, RoleProcurement:
		return true
	}
	return false
}

func (s *Service) Create(ctx context.Context, dto CreateUserDTO) (*User, error) {
	dto.Email = strings.ToLower(strings.TrimSpace(dto.Email))
	if dto.Name == "" || dto.Email == "" || dto.Password == "" {
		return nil, errors.New("name, email and password are required")
	}
	if _, err := mail.ParseAddress(dto.Email); err != nil {
		return nil, errors.New("invalid email address")
	}
	if len(dto.Password) < minPasswordLen {
		return nil, fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}

	role := dto.Role
	if role == "" {
		role = RoleStaff
	}
	if !role.Valid() {
		return nil, errors.New("invalid role")
	}

	hash, err := utils.HashPassword(dto.Password)
	if err != nil {
		return nil, err
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
		if strings.Contains(err.Error(), "23505") {
			return nil, errors.New("a user with this email already exists")
		}
		return nil, err
	}
	s.departmentChanged(ctx, nil, u)
	return u, nil
}

// GetByID returns a user, served from cache when possible. The cached copy never
// contains the password hash (it is excluded from JSON), so never use it for
// authentication — use GetForLogin instead.
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

// GetForLogin always reads from the database so the password hash is present.
func (s *Service) GetForLogin(ctx context.Context, email string) (*User, error) {
	return s.repo.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
}

// Identity implements middleware.IdentityLookup.
func (s *Service) Identity(ctx context.Context, id string) (*middleware.Identity, error) {
	u, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	dept := ""
	if u.DepartmentID != nil {
		dept = *u.DepartmentID
	}
	return &middleware.Identity{Role: string(u.Role), DepartmentID: dept, Active: u.IsActive}, nil
}

func (s *Service) List(ctx context.Context) ([]*User, error) {
	return s.repo.List(ctx)
}

func (s *Service) ListByDepartment(ctx context.Context, deptID string) ([]*User, error) {
	return s.repo.ListByDepartment(ctx, deptID)
}

var ErrValidation = errors.New("validation failed")

func (s *Service) Update(ctx context.Context, id string, dto UpdateUserDTO) (*User, error) {
	if dto.Role != nil && !dto.Role.Valid() {
		return nil, fmt.Errorf("%w: invalid role", ErrValidation)
	}
	if dto.Name != nil && strings.TrimSpace(*dto.Name) == "" {
		return nil, fmt.Errorf("%w: name cannot be empty", ErrValidation)
	}
	if dto.Email != nil {
		e := strings.ToLower(strings.TrimSpace(*dto.Email))
		if _, err := mail.ParseAddress(e); err != nil || e == "" {
			return nil, fmt.Errorf("%w: invalid email address", ErrValidation)
		}
		dto.Email = &e
	}
	// Remember the previous department and status to detect department changes.
	before, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	u, err := s.repo.Update(ctx, id, dto)
	if err != nil {
		if strings.Contains(err.Error(), "23505") {
			return nil, fmt.Errorf("%w: a user with this email already exists", ErrValidation)
		}
		return nil, err
	}
	cache.Delete(ctx, s.cache, userKey(id))
	s.departmentChanged(ctx, before, u)
	return u, nil
}

// UpdateMe applies the profile fields any user may change on their own account.
func (s *Service) UpdateMe(ctx context.Context, id string, dto UpdateMeDTO) (*User, error) {
	return s.Update(ctx, id, UpdateUserDTO{Name: dto.Name, Email: dto.Email, Position: dto.Position})
}

var ErrWrongPassword = errors.New("current password is incorrect")

// ChangePassword verifies the current password before setting a new one.
func (s *Service) ChangePassword(ctx context.Context, id string, dto ChangePasswordDTO) error {
	if len(dto.NewPassword) < minPasswordLen {
		return fmt.Errorf("%w: password must be at least %d characters", ErrValidation, minPasswordLen)
	}
	u, err := s.repo.FindByID(ctx, id) // from the DB: the cached copy has no hash
	if err != nil {
		return err
	}
	if !utils.CheckPassword(dto.CurrentPassword, u.PasswordHash) {
		return ErrWrongPassword
	}
	hash, err := utils.HashPassword(dto.NewPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, id, hash)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	cache.Delete(ctx, s.cache, userKey(id))
	return nil
}

// EnsureSuperAdmin creates the first super_admin from configuration when the
// database has none. It does nothing if a super_admin already exists.
func (s *Service) EnsureSuperAdmin(ctx context.Context, email, password string) error {
	admins, err := s.repo.ListByRole(ctx, string(RoleSuperAdmin))
	if err != nil {
		return err
	}
	if len(admins) > 0 {
		return nil
	}
	if email == "" || password == "" {
		log.Println("WARNING: no super_admin exists; set BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD to create one")
		return nil
	}
	_, err = s.Create(ctx, CreateUserDTO{Name: "Admin", Email: email, Password: password, Role: RoleSuperAdmin})
	if err == nil {
		log.Println("created bootstrap super_admin account")
	}
	return err
}

// DepartmentOf returns a user's department ID ("" if none). It implements
// middleware.UserDepartmentLookup.
func (s *Service) DepartmentOf(ctx context.Context, id string) (string, error) {
	u, err := s.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if u.DepartmentID == nil {
		return "", nil
	}
	return *u.DepartmentID, nil
}
