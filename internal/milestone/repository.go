package milestone

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const cols = `id, project_id, title, to_char(due_date, 'YYYY-MM-DD'), achieved_at, created_by, created_at`

func scan(row interface{ Scan(...any) error }) (*Milestone, error) {
	m := &Milestone{}
	err := row.Scan(&m.ID, &m.ProjectID, &m.Title, &m.DueDate, &m.AchievedAt, &m.CreatedBy, &m.CreatedAt)
	return m, err
}

func (r *Repository) Create(ctx context.Context, m *Milestone) error {
	q := `INSERT INTO project_milestones (project_id, title, due_date, created_by)
	      VALUES ($1,$2,$3::date,$4) RETURNING ` + cols
	got, err := scan(r.db.QueryRow(ctx, q, m.ProjectID, m.Title, m.DueDate, m.CreatedBy))
	if err != nil {
		return err
	}
	*m = *got
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Milestone, error) {
	return scan(r.db.QueryRow(ctx, `SELECT `+cols+` FROM project_milestones WHERE id=$1`, id))
}

func (r *Repository) ListByProject(ctx context.Context, projectID string) ([]*Milestone, error) {
	rows, err := r.db.Query(ctx, `SELECT `+cols+` FROM project_milestones WHERE project_id=$1 ORDER BY due_date ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Milestone
	for rows.Next() {
		m, err := scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, nil
}

func (r *Repository) Update(ctx context.Context, id string, dto UpdateMilestoneDTO) (*Milestone, error) {
	m, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dto.Title != nil {
		m.Title = *dto.Title
	}
	if dto.DueDate != nil {
		m.DueDate = *dto.DueDate
	}
	_, err = r.db.Exec(ctx, `UPDATE project_milestones SET title=$1, due_date=$2::date WHERE id=$3`, m.Title, m.DueDate, id)
	return m, err
}

func (r *Repository) Achieve(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `UPDATE project_milestones SET achieved_at=$1 WHERE id=$2`, now, id)
	return err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM project_milestones WHERE id=$1`, id)
	return err
}
