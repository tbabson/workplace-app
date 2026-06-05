package review

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const cols = `id, user_id, reviewer_id, period, rating, goals, comments, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (*Review, error) {
	rv := &Review{}
	return rv, row.Scan(&rv.ID, &rv.UserID, &rv.ReviewerID, &rv.Period, &rv.Rating, &rv.Goals, &rv.Comments, &rv.CreatedAt, &rv.UpdatedAt)
}

func (r *Repository) Create(ctx context.Context, rv *Review) error {
	q := `INSERT INTO performance_reviews (user_id, reviewer_id, period, rating, goals, comments)
	      VALUES ($1,$2,$3,$4,$5,$6) RETURNING ` + cols
	got, err := scan(r.db.QueryRow(ctx, q, rv.UserID, rv.ReviewerID, rv.Period, rv.Rating, rv.Goals, rv.Comments))
	if err != nil {
		return err
	}
	*rv = *got
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Review, error) {
	return scan(r.db.QueryRow(ctx, `SELECT `+cols+` FROM performance_reviews WHERE id=$1`, id))
}

func (r *Repository) List(ctx context.Context, userID string) ([]*Review, error) {
	var q string
	var args []any
	if userID != "" {
		q = `SELECT ` + cols + ` FROM performance_reviews WHERE user_id=$1 ORDER BY created_at DESC`
		args = []any{userID}
	} else {
		q = `SELECT ` + cols + ` FROM performance_reviews ORDER BY created_at DESC`
	}
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Review
	for rows.Next() {
		rv, err := scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, rv)
	}
	return list, nil
}

func (r *Repository) Update(ctx context.Context, id string, dto UpdateReviewDTO) (*Review, error) {
	rv, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dto.Period != nil {
		rv.Period = *dto.Period
	}
	if dto.Rating != nil {
		rv.Rating = *dto.Rating
	}
	if dto.Goals != nil {
		rv.Goals = *dto.Goals
	}
	if dto.Comments != nil {
		rv.Comments = *dto.Comments
	}
	rv.UpdatedAt = time.Now()
	_, err = r.db.Exec(ctx, `UPDATE performance_reviews SET period=$1, rating=$2, goals=$3, comments=$4, updated_at=$5 WHERE id=$6`,
		rv.Period, rv.Rating, rv.Goals, rv.Comments, rv.UpdatedAt, id)
	return rv, err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM performance_reviews WHERE id=$1`, id)
	return err
}
