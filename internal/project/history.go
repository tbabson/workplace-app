package project

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HistoryEntry struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	ChangedBy string    `json:"changed_by"`
	Field     string    `json:"field"`
	FromValue *string   `json:"from_value"`
	ToValue   string    `json:"to_value"`
	CreatedAt time.Time `json:"created_at"`
}

type HistoryRepository struct{ db *pgxpool.Pool }

func NewHistoryRepository(db *pgxpool.Pool) *HistoryRepository { return &HistoryRepository{db: db} }

func (r *HistoryRepository) Log(ctx context.Context, e *HistoryEntry) error {
	q := `INSERT INTO project_history (project_id, changed_by, field, from_value, to_value)
	      VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at`
	return r.db.QueryRow(ctx, q, e.ProjectID, e.ChangedBy, e.Field, e.FromValue, e.ToValue).
		Scan(&e.ID, &e.CreatedAt)
}

func (r *HistoryRepository) List(ctx context.Context, projectID string) ([]*HistoryEntry, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, changed_by, field, from_value, to_value, created_at
		 FROM project_history WHERE project_id=$1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*HistoryEntry
	for rows.Next() {
		e := &HistoryEntry{}
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.ChangedBy, &e.Field, &e.FromValue, &e.ToValue, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, nil
}
