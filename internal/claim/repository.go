package claim

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const cols = `id, user_id, title, amount, to_char("date", 'YYYY-MM-DD'), description, receipt_url, status, reviewed_by, reviewed_at, paid_at, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (*Claim, error) {
	c := &Claim{}
	return c, row.Scan(&c.ID, &c.UserID, &c.Title, &c.Amount, &c.Date, &c.Description,
		&c.ReceiptURL, &c.Status, &c.ReviewedBy, &c.ReviewedAt, &c.PaidAt, &c.CreatedAt, &c.UpdatedAt)
}

func (r *Repository) Create(ctx context.Context, c *Claim) error {
	q := `INSERT INTO expense_claims (user_id, title, amount, "date", description, receipt_url)
	      VALUES ($1,$2,$3,$4::date,$5,$6) RETURNING ` + cols
	got, err := scan(r.db.QueryRow(ctx, q, c.UserID, c.Title, c.Amount, c.Date, c.Description, c.ReceiptURL))
	if err != nil {
		return err
	}
	*c = *got
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Claim, error) {
	return scan(r.db.QueryRow(ctx, `SELECT `+cols+` FROM expense_claims WHERE id=$1`, id))
}

func (r *Repository) List(ctx context.Context, userID string) ([]*Claim, error) {
	var q string
	var args []any
	if userID != "" {
		q = `SELECT ` + cols + ` FROM expense_claims WHERE user_id=$1 ORDER BY created_at DESC`
		args = []any{userID}
	} else {
		q = `SELECT ` + cols + ` FROM expense_claims ORDER BY created_at DESC`
	}
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Claim
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status Status, reviewerID string) (*Claim, error) {
	now := time.Now()
	_, err := r.db.Exec(ctx,
		`UPDATE expense_claims SET status=$1, reviewed_by=$2, reviewed_at=$3, updated_at=$3 WHERE id=$4`,
		status, reviewerID, now, id)
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *Repository) MarkPaid(ctx context.Context, id string, reviewerID string) (*Claim, error) {
	now := time.Now()
	_, err := r.db.Exec(ctx,
		`UPDATE expense_claims SET status=$1, paid_at=$2, reviewed_by=$3, reviewed_at=$2, updated_at=$2 WHERE id=$4`,
		StatusPaid, now, reviewerID, id)
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM expense_claims WHERE id=$1`, id)
	return err
}
