package attendance

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

const cols = `id, user_id, sign_in_at, sign_out_at, sign_in_lat, sign_in_lng, date::text, auto_closed, device_id, created_at`

func scan(a *Attendance, row interface{ Scan(...any) error }) error {
	return row.Scan(
		&a.ID, &a.UserID, &a.SignInAt, &a.SignOutAt,
		&a.SignInLat, &a.SignInLng, &a.Date, &a.AutoClosed, &a.DeviceID, &a.CreatedAt,
	)
}

func (r *Repository) SignIn(ctx context.Context, userID, deviceID string, lat, lng float64) (*Attendance, error) {
	now := time.Now()
	a := &Attendance{}
	q := `INSERT INTO attendance (user_id, sign_in_at, sign_in_lat, sign_in_lng, date, device_id)
	      VALUES ($1, $2, $3, $4, CURRENT_DATE, $5)
	      RETURNING ` + cols
	return a, scan(a, r.db.QueryRow(ctx, q, userID, now, lat, lng, deviceID))
}

func (r *Repository) SignOut(ctx context.Context, userID string) (*Attendance, error) {
	now := time.Now()
	a := &Attendance{}
	q := `UPDATE attendance SET sign_out_at = $1
	      WHERE user_id = $2 AND date = CURRENT_DATE AND sign_in_at IS NOT NULL AND sign_out_at IS NULL
	      RETURNING ` + cols
	return a, scan(a, r.db.QueryRow(ctx, q, now, userID))
}

func (r *Repository) TodayRecord(ctx context.Context, userID string) (*Attendance, error) {
	a := &Attendance{}
	return a, scan(a, r.db.QueryRow(ctx,
		`SELECT `+cols+` FROM attendance WHERE user_id = $1 AND date = CURRENT_DATE`, userID,
	))
}

// IsDeviceSignedInToday returns true if the device has already been used to sign
// in a DIFFERENT user today. An empty deviceID always returns false.
func (r *Repository) IsDeviceSignedInToday(ctx context.Context, deviceID, userID string) (bool, error) {
	if deviceID == "" {
		return false, nil
	}
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(
		   SELECT 1 FROM attendance
		   WHERE date = CURRENT_DATE
		     AND sign_in_at IS NOT NULL
		     AND device_id = $1
		     AND user_id != $2
		 )`,
		deviceID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *Repository) ListSignedInToday(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT user_id FROM attendance
		 WHERE date = CURRENT_DATE AND sign_in_at IS NOT NULL AND sign_out_at IS NULL`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID string) ([]*Attendance, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+cols+` FROM attendance WHERE user_id = $1 ORDER BY date DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Attendance
	for rows.Next() {
		a := &Attendance{}
		if err := scan(a, rows); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

func (r *Repository) ListAll(ctx context.Context) ([]*Attendance, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+cols+` FROM attendance ORDER BY date DESC, created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Attendance
	for rows.Next() {
		a := &Attendance{}
		if err := scan(a, rows); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

// AutoCloseOpenRecords sets sign_out_at and marks auto_closed=true for all records
// that are still open (signed in but not signed out) on the current day.
// Returns the user IDs of affected records for notification purposes.
func (r *Repository) AutoCloseOpenRecords(ctx context.Context, signOutAt time.Time) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`UPDATE attendance
		 SET sign_out_at = $1, auto_closed = true
		 WHERE date = CURRENT_DATE AND sign_in_at IS NOT NULL AND sign_out_at IS NULL
		 RETURNING user_id`,
		signOutAt,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		ids = append(ids, uid)
	}
	return ids, nil
}

// AdminSignOut allows an admin to close an open record with a specific sign-out time.
func (r *Repository) AdminSignOut(ctx context.Context, id string, signOutAt time.Time) (*Attendance, error) {
	a := &Attendance{}
	q := `UPDATE attendance SET sign_out_at = $1
	      WHERE id = $2 AND sign_in_at IS NOT NULL AND sign_out_at IS NULL
	      RETURNING ` + cols
	return a, scan(a, r.db.QueryRow(ctx, q, signOutAt, id))
}
