package attendance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"workplace/internal/notification"
	"workplace/internal/user"
	"workplace/pkg/utils"
)

type Service struct {
	repo              *Repository
	notifSvc          *notification.Service
	userRepo          *user.Repository
	companyLat        float64
	companyLng        float64
	geofenceRadius    float64
	deviceLockEnabled bool
}

func NewService(repo *Repository, notifSvc *notification.Service, userRepo *user.Repository, companyLat, companyLng, geofenceRadius float64, deviceLockEnabled bool) *Service {
	return &Service{
		repo:              repo,
		notifSvc:          notifSvc,
		userRepo:          userRepo,
		companyLat:        companyLat,
		companyLng:        companyLng,
		geofenceRadius:    geofenceRadius,
		deviceLockEnabled: deviceLockEnabled,
	}
}

func (s *Service) SignIn(ctx context.Context, userID string, dto SignInDTO) (*Attendance, error) {
	if dto.Latitude < -90 || dto.Latitude > 90 || dto.Longitude < -180 || dto.Longitude > 180 {
		return nil, errors.New("invalid coordinates")
	}
	// Note: coordinates come from the client, so the geofence is a deterrent,
	// not proof of presence.
	if !utils.IsWithinGeofence(dto.Latitude, dto.Longitude, s.companyLat, s.companyLng, s.geofenceRadius) {
		return nil, errors.New("you are not within the company premises")
	}

	// Prevent double sign-in
	existing, err := s.repo.TodayRecord(ctx, userID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil && existing.SignInAt != nil {
		return nil, errors.New("you have already signed in today")
	}

	// Device lock: block if this device already signed in a different user today
	if s.deviceLockEnabled {
		if dto.DeviceID == "" {
			return nil, errors.New("device_id is required")
		}
		taken, err := s.repo.IsDeviceSignedInToday(ctx, dto.DeviceID, userID)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, errors.New("this device has already been used to sign in today")
		}
	}

	a, err := s.repo.SignIn(ctx, userID, dto.DeviceID, dto.Latitude, dto.Longitude)
	if err != nil {
		if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key") {
			return nil, errors.New("you have already signed in today")
		}
		return nil, err
	}

	go s.notifySignEvent(userID, true)
	return a, nil
}

func (s *Service) SignOut(ctx context.Context, userID string) (*Attendance, error) {
	a, err := s.repo.SignOut(ctx, userID)
	if err != nil {
		return nil, err
	}
	go s.notifySignEvent(userID, false)
	return a, nil
}

// notifySignEvent alerts super_admins about any sign-in/out, and alerts the
// user's dept_head when a staff or procurement member signs in/out.
func (s *Service) notifySignEvent(userID string, signIn bool) {
	ctx := context.Background()

	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return
	}

	action := "signed in"
	notifType := notification.TypeAttendanceSignIn
	if !signIn {
		action = "signed out"
		notifType = notification.TypeAttendanceSignOut
	}

	position := ""
	if u.Position != nil {
		position = *u.Position
	}
	title := fmt.Sprintf("%s has %s", u.Name, action)
	body := fmt.Sprintf("%s (%s) has %s.", u.Name, position, action)

	// Notify all super_admins
	admins, err := s.userRepo.ListByRole(ctx, "super_admin")
	if err == nil {
		var inputs []notification.NotifyInput
		for _, a := range admins {
			if a.ID == userID {
				continue
			}
			inputs = append(inputs, notification.NotifyInput{
				UserID: a.ID,
				Type:   notifType,
				Title:  title,
				Body:   body,
			})
		}
		s.notifSvc.NotifyMany(ctx, inputs)
	}

	// Notify the dept_head of the user's department (only for staff/procurement)
	if (u.Role == "staff" || u.Role == "procurement") && u.DepartmentID != nil {
		deptMembers, err := s.userRepo.ListByDepartment(ctx, *u.DepartmentID)
		if err == nil {
			for _, m := range deptMembers {
				if m.Role == "dept_head" && m.ID != userID {
					s.notifSvc.Notify(ctx, notification.NotifyInput{
						UserID: m.ID,
						Type:   notifType,
						Title:  title,
						Body:   body,
					})
					break
				}
			}
		}
	}
}

func (s *Service) MyHistory(ctx context.Context, userID string) ([]*Attendance, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *Service) UserHistory(ctx context.Context, userID string) ([]*Attendance, error) {
	return s.repo.ListByUser(ctx, userID)
}

// AllHistory returns every record, or only those of users in deptID when set.
func (s *Service) AllHistory(ctx context.Context, deptID string) ([]*Attendance, error) {
	return s.repo.ListAll(ctx, deptID)
}

// Get returns one attendance record.
func (s *Service) Get(ctx context.Context, id string) (*Attendance, error) {
	return s.repo.FindByID(ctx, id)
}

// AdminSignOut lets a super_admin or dept_head manually close an open attendance record.
// signOutAt is optional — defaults to time.Now() if nil.
func (s *Service) AdminSignOut(ctx context.Context, id string, signOutAt *time.Time) (*Attendance, error) {
	t := time.Now()
	if signOutAt != nil {
		t = *signOutAt
	}
	a, err := s.repo.AdminSignOut(ctx, id, t)
	if err != nil {
		return nil, errors.New("record not found or already signed out")
	}
	return a, nil
}
