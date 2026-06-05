package task

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const cols = `id, project_id, parent_id, title, description, status, assigned_to, to_char(due_date, 'YYYY-MM-DD'), created_by, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (*Task, error) {
	t := &Task{}
	err := row.Scan(&t.ID, &t.ProjectID, &t.ParentID, &t.Title, &t.Description,
		&t.Status, &t.AssignedTo, &t.DueDate, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (r *Repository) Create(ctx context.Context, t *Task) error {
	q := `INSERT INTO project_tasks (project_id, parent_id, title, description, assigned_to, due_date, created_by)
	      VALUES ($1,$2,$3,$4,$5,$6::date,$7) RETURNING ` + cols
	got, err := scan(r.db.QueryRow(ctx, q, t.ProjectID, t.ParentID, t.Title, t.Description, t.AssignedTo, t.DueDate, t.CreatedBy))
	if err != nil {
		return err
	}
	*t = *got
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Task, error) {
	return scan(r.db.QueryRow(ctx, `SELECT `+cols+` FROM project_tasks WHERE id=$1`, id))
}

func (r *Repository) ListByProject(ctx context.Context, projectID string) ([]*Task, error) {
	rows, err := r.db.Query(ctx, `SELECT `+cols+` FROM project_tasks WHERE project_id=$1 ORDER BY created_at ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []*Task
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (r *Repository) Update(ctx context.Context, id string, dto UpdateTaskDTO) (*Task, error) {
	t, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dto.Title != nil {
		t.Title = *dto.Title
	}
	if dto.Description != nil {
		t.Description = *dto.Description
	}
	if dto.Status != nil {
		t.Status = *dto.Status
	}
	if dto.AssignedTo != nil {
		t.AssignedTo = dto.AssignedTo
	}
	if dto.DueDate != nil {
		t.DueDate = dto.DueDate
	}
	t.UpdatedAt = time.Now()
	q := `UPDATE project_tasks SET title=$1, description=$2, status=$3, assigned_to=$4, due_date=$5::date, updated_at=$6 WHERE id=$7`
	_, err = r.db.Exec(ctx, q, t.Title, t.Description, t.Status, t.AssignedTo, t.DueDate, t.UpdatedAt, id)
	return t, err
}

func (r *Repository) ListDueTomorrow(ctx context.Context) ([]*Task, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+cols+` FROM project_tasks
		 WHERE due_date = CURRENT_DATE + 1
		 AND assigned_to IS NOT NULL
		 AND status != 'done'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []*Task
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM project_tasks WHERE id=$1`, id)
	return err
}
