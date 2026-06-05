package query

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

const cols = `id, title, content, issuer_id, recipient_id, status, response, responded_at, created_at, updated_at`

func scanQuery(row interface{ Scan(...interface{}) error }) (*Query, error) {
	q := &Query{}
	return q, row.Scan(
		&q.ID, &q.Title, &q.Content, &q.IssuerID, &q.RecipientID,
		&q.Status, &q.Response, &q.RespondedAt, &q.CreatedAt, &q.UpdatedAt,
	)
}

func (r *Repository) Create(ctx context.Context, q *Query) error {
	sql := `INSERT INTO queries (title, content, issuer_id, recipient_id)
	        VALUES ($1, $2, $3, $4)
	        RETURNING id, status, response, responded_at, created_at, updated_at`
	return r.db.QueryRow(ctx, sql, q.Title, q.Content, q.IssuerID, q.RecipientID).
		Scan(&q.ID, &q.Status, &q.Response, &q.RespondedAt, &q.CreatedAt, &q.UpdatedAt)
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Query, error) {
	return scanQuery(r.db.QueryRow(ctx, `SELECT `+cols+` FROM queries WHERE id = $1`, id))
}

func (r *Repository) ListForUser(ctx context.Context, userID string) ([]*Query, error) {
	sql := `SELECT ` + cols + ` FROM queries WHERE recipient_id = $1 OR issuer_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, sql, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Query
	for rows.Next() {
		q, err := scanQuery(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, q)
	}
	return list, nil
}

func (r *Repository) ListAll(ctx context.Context) ([]*Query, error) {
	rows, err := r.db.Query(ctx, `SELECT `+cols+` FROM queries ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Query
	for rows.Next() {
		q, err := scanQuery(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, q)
	}
	return list, nil
}

func (r *Repository) Respond(ctx context.Context, id string, dto RespondDTO) (*Query, error) {
	now := time.Now()
	sql := `UPDATE queries SET status=$1, response=$2, responded_at=$3, updated_at=$4
	        WHERE id=$5
	        RETURNING ` + cols
	return scanQuery(r.db.QueryRow(ctx, sql, dto.Status, dto.Response, now, now, id))
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM queries WHERE id = $1`, id)
	return err
}
