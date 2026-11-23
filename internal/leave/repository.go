package leave

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const cols = `id, user_id, type, to_char(start_date, 'YYYY-MM-DD'), to_char(end_date, 'YYYY-MM-DD'), days, reason, status, reviewed_by, reviewed_at, review_note, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (*Request, error) {
	req := &Request{}
	return req, row.Scan(&req.ID, &req.UserID, &req.Type, &req.StartDate, &req.EndDate,
		&req.Days, &req.Reason, &req.Status, &req.ReviewedBy, &req.ReviewedAt, &req.ReviewNote,
		&req.CreatedAt, &req.UpdatedAt)
}

func (r *Repository) Create(ctx context.Context, req *Request) error {
	q := `INSERT INTO leave_requests (user_id, type, start_date, end_date, days, reason)
	      VALUES ($1,$2,$3::date,$4::date,$5,$6) RETURNING ` + cols
	got, err := scan(r.db.QueryRow(ctx, q, req.UserID, req.Type, req.StartDate, req.EndDate, req.Days, req.Reason))
	if err != nil {
		return err
	}
	*req = *got
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Request, error) {
	return scan(r.db.QueryRow(ctx, `SELECT `+cols+` FROM leave_requests WHERE id=$1`, id))
}

func (r *Repository) List(ctx context.Context, userID string) ([]*Request, error) {
	var q string
	var args []any
	if userID != "" {
		q = `SELECT ` + cols + ` FROM leave_requests WHERE user_id=$1 ORDER BY created_at DESC`
		args = []any{userID}
	} else {
		q = `SELECT ` + cols + ` FROM leave_requests ORDER BY created_at DESC`
	}
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Request
	for rows.Next() {
		req, err := scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, req)
	}
	return list, nil
}

func (r *Repository) Review(ctx context.Context, id, reviewerID string, status Status, note *string) error {
	now := time.Now()
	tag, err := r.db.Exec(ctx,
		`UPDATE leave_requests SET status=$1, reviewed_by=$2, reviewed_at=$3, review_note=$4, updated_at=$3
		 WHERE id=$5 AND status='pending'`,
		status, reviewerID, now, note, id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

// Approve marks a pending request approved and deducts the days from the
// balance for the leave's year in one transaction, so the two never diverge.
func (r *Repository) Approve(ctx context.Context, req *Request, reviewerID string, note *string, year int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	tag, err := tx.Exec(ctx,
		`UPDATE leave_requests SET status=$1, reviewed_by=$2, reviewed_at=$3, review_note=$4, updated_at=$3
		 WHERE id=$5 AND status='pending'`,
		StatusApproved, reviewerID, now, note, req.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if _, err := tx.Exec(ctx,
		`UPDATE leave_balances SET used = used + $1 WHERE user_id=$2 AND type=$3 AND year=$4`,
		req.Days, req.UserID, req.Type, year); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) Cancel(ctx context.Context, id, userID string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE leave_requests SET status='cancelled', updated_at=$1 WHERE id=$2 AND user_id=$3 AND status='pending'`,
		time.Now(), id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

func (r *Repository) GetBalance(ctx context.Context, userID string, year int) ([]*Balance, error) {
	rows, err := r.db.Query(ctx,
		`SELECT user_id, type, year, total, used FROM leave_balances WHERE user_id=$1 AND year=$2`, userID, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Balance
	for rows.Next() {
		b := &Balance{}
		if err := rows.Scan(&b.UserID, &b.Type, &b.Year, &b.Total, &b.Used); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, nil
}

func (r *Repository) SetBalance(ctx context.Context, b *Balance) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO leave_balances (user_id, type, year, total, used) VALUES ($1,$2,$3,$4,0)
		 ON CONFLICT (user_id, type, year) DO UPDATE SET total=$4`,
		b.UserID, b.Type, b.Year, b.Total)
	return err
}
