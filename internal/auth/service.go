package auth

import (
	"context"
	"errors"
	"log"
	"time"

	"workplace/internal/user"
	"workplace/pkg/utils"

	"github.com/redis/go-redis/v9"
)

type Service struct {
	userSvc       *user.Service
	secret        string
	expirationHrs int
	rdb           *redis.Client
}

func NewService(userSvc *user.Service, secret string, expHrs int, rdb *redis.Client) *Service {
	return &Service{
		userSvc:       userSvc,
		secret:        secret,
		expirationHrs: expHrs,
		rdb:           rdb,
	}
}

func (s *Service) Login(ctx context.Context, dto LoginDTO) (string, error) {
	u, err := s.userSvc.GetByEmail(ctx, dto.Email)
	if err != nil {
		log.Printf("login: lookup failed for %s: %v", dto.Email, err)
		return "", errors.New("invalid credentials")
	}

	if !u.IsActive {
		return "", errors.New("account is deactivated")
	}

	if !utils.CheckPassword(dto.Password, u.PasswordHash) {
		return "", errors.New("invalid credentials")
	}

	deptID := ""
	if u.DepartmentID != nil {
		deptID = *u.DepartmentID
	}

	return utils.GenerateToken(u.ID, string(u.Role), deptID, s.secret, s.expirationHrs)
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.rdb.Set(ctx, "blacklist:"+token, "1",
		time.Duration(s.expirationHrs)*time.Hour).Err()
}
