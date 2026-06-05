package notification

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, n *Notification) error {
	q := `INSERT INTO notifications (user_id, type, title, body, ref_id)
	      VALUES ($1, $2, $3, $4, $5)
	      RETURNING id, is_read, created_at`
	return r.db.QueryRow(ctx, q, n.UserID, n.Type, n.Title, n.Body, n.RefID).
		Scan(&n.ID, &n.IsRead, &n.CreatedAt)
}

func (r *Repository) ListForUser(ctx context.Context, userID string) ([]*Notification, error) {
	q := `SELECT id, user_id, type, title, body, ref_id, is_read, created_at
	      FROM notifications WHERE user_id = $1
	      ORDER BY created_at DESC LIMIT 50`
	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Notification
	for rows.Next() {
		n := &Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body,
			&n.RefID, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	return list, nil
}

func (r *Repository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(1) FROM notifications WHERE user_id=$1 AND is_read=false`, userID,
	).Scan(&count)
	return count, err
}

func (r *Repository) MarkRead(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE notifications SET is_read=true WHERE id=$1 AND user_id=$2`, id, userID,
	)
	return err
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE notifications SET is_read=true WHERE user_id=$1 AND is_read=false`,
		userID,
	)
	return err
}

func (r *Repository) ClearAll(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM notifications WHERE user_id=$1`, userID)
	return err
}
