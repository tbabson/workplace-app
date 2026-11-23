package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"strings"
	"time"

	"workplace/internal/user"
	"workplace/pkg/utils"

	"github.com/redis/go-redis/v9"
)

const (
	maxFailedLogins = 10
	failWindow      = 15 * time.Minute
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTooManyAttempts    = errors.New("too many failed login attempts; try again later")
)

// dummyHash is compared against when the email doesn't exist, so a login for an
// unknown email takes as long as one for a real account.
var dummyHash, _ = utils.HashPassword("dummy-password-for-timing")

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

// TokenTTL is how long issued tokens (and the auth cookie) are valid.
func (s *Service) TokenTTL() time.Duration {
	return time.Duration(s.expirationHrs) * time.Hour
}

func failKey(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return "login:fail:" + hex.EncodeToString(sum[:])
}

func (s *Service) Login(ctx context.Context, dto LoginDTO) (string, error) {
	key := failKey(dto.Email)
	if n, err := s.rdb.Get(ctx, key).Int(); err == nil && n >= maxFailedLogins {
		return "", ErrTooManyAttempts
	}

	u, err := s.userSvc.GetForLogin(ctx, dto.Email)
	if err != nil {
		utils.CheckPassword(dto.Password, dummyHash)
		s.recordFailure(ctx, key)
		return "", ErrInvalidCredentials
	}

	// Check the password before anything else so the response never reveals
	// whether an account exists or is deactivated to someone without the password.
	if !utils.CheckPassword(dto.Password, u.PasswordHash) {
		s.recordFailure(ctx, key)
		return "", ErrInvalidCredentials
	}

	if !u.IsActive {
		return "", errors.New("account is deactivated")
	}

	s.rdb.Del(ctx, key)

	deptID := ""
	if u.DepartmentID != nil {
		deptID = *u.DepartmentID
	}

	return utils.GenerateToken(u.ID, string(u.Role), deptID, s.secret, s.expirationHrs)
}

func (s *Service) recordFailure(ctx context.Context, key string) {
	pipe := s.rdb.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, failWindow)
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("login: record failed attempt: %v", err)
	}
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.rdb.Set(ctx, "blacklist:"+token, "1", s.TokenTTL()).Err()
}
