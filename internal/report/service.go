package report

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"workplace/pkg/cache"
)

const ttlDashboard = 30 * time.Second

type Service struct {
	repo  *Repository
	cache *cache.Cache
}

func NewService(repo *Repository, c *cache.Cache) *Service { return &Service{repo: repo, cache: c} }

func defaultDateRange(from, to string) (string, string) {
	if from == "" {
		from = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}
	return from, to
}

func (s *Service) Dashboard(ctx context.Context, userID, role, deptID string) (*DashboardData, error) {
	cacheKey := fmt.Sprintf("report:dashboard:%s:%s", role, deptID)
	if d, ok := cache.Get[DashboardData](ctx, s.cache, cacheKey); ok {
		return d, nil
	}
	d, err := s.repo.Dashboard(ctx, userID, role, deptID)
	if err != nil {
		return nil, err
	}
	cache.Set(ctx, s.cache, cacheKey, d, ttlDashboard)
	return d, nil
}

func (s *Service) AttendanceReport(ctx context.Context, from, to string) ([]*AttendanceRow, error) {
	from, to = defaultDateRange(from, to)
	return s.repo.AttendanceReport(ctx, from, to)
}

func (s *Service) AttendanceCSV(ctx context.Context, from, to string) ([]byte, error) {
	rows, err := s.AttendanceReport(ctx, from, to)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"Date", "User ID", "User Name", "Sign In", "Sign Out", "Hours Worked"})
	for _, row := range rows {
		signIn, signOut := "", ""
		if row.SignInAt != nil {
			signIn = row.SignInAt.Format(time.RFC3339)
		}
		if row.SignOutAt != nil {
			signOut = row.SignOutAt.Format(time.RFC3339)
		}
		_ = w.Write([]string{row.Date, row.UserID, row.UserName, signIn, signOut, fmt.Sprintf("%.2f", row.HoursWorked)})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func (s *Service) OvertimeReport(ctx context.Context, from, to string) ([]*OvertimeRow, error) {
	from, to = defaultDateRange(from, to)
	return s.repo.OvertimeReport(ctx, from, to, 8.0)
}

func (s *Service) OvertimeCSV(ctx context.Context, from, to string) ([]byte, error) {
	rows, err := s.OvertimeReport(ctx, from, to)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"Date", "User ID", "User Name", "Overtime Hours"})
	for _, row := range rows {
		_ = w.Write([]string{row.Date, row.UserID, row.UserName, fmt.Sprintf("%.2f", row.OvertimeHours)})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
