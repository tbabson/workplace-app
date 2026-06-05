package project

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

const selectCols = `id, title, description, status, progress,
                    to_char(start_date, 'YYYY-MM-DD'), to_char(end_date, 'YYYY-MM-DD'),
                    department_id, created_by, created_at, updated_at`

func scanProject(row pgx.Row) (*Project, error) {
	p := &Project{}
	err := row.Scan(
		&p.ID, &p.Title, &p.Description, &p.Status, &p.Progress,
		&p.StartDate, &p.EndDate, &p.DepartmentID, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func (r *Repository) Create(ctx context.Context, p *Project) error {
	q := `INSERT INTO projects (title, description, status, start_date, end_date, department_id, created_by)
	      VALUES ($1, $2, $3, $4::date, $5::date, $6, $7)
	      RETURNING ` + selectCols
	row := r.db.QueryRow(ctx, q,
		p.Title, p.Description, p.Status, p.StartDate, p.EndDate, p.DepartmentID, p.CreatedBy,
	)
	got, err := scanProject(row)
	if err != nil {
		return err
	}
	*p = *got
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Project, error) {
	return scanProject(r.db.QueryRow(ctx,
		`SELECT `+selectCols+` FROM projects WHERE id = $1`, id,
	))
}

func (r *Repository) List(ctx context.Context, deptID string) ([]*Project, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if deptID != "" {
		rows, err = r.db.Query(ctx,
			`SELECT `+selectCols+` FROM projects WHERE department_id = $1 ORDER BY created_at DESC`, deptID)
	} else {
		rows, err = r.db.Query(ctx,
			`SELECT `+selectCols+` FROM projects ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		p := &Project{}
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Status, &p.Progress,
			&p.StartDate, &p.EndDate, &p.DepartmentID, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (r *Repository) Update(ctx context.Context, id string, dto UpdateProjectDTO) (*Project, error) {
	p, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if dto.Title != nil {
		p.Title = *dto.Title
	}
	if dto.Description != nil {
		p.Description = *dto.Description
	}
	if dto.Status != nil {
		p.Status = *dto.Status
	}
	if dto.StartDate != nil {
		p.StartDate = dto.StartDate
	}
	if dto.EndDate != nil {
		p.EndDate = dto.EndDate
	}
	if dto.DepartmentID != nil {
		p.DepartmentID = dto.DepartmentID
	}
	p.UpdatedAt = time.Now()

	q := `UPDATE projects SET title=$1, description=$2, status=$3, start_date=$4::date,
	      end_date=$5::date, department_id=$6, updated_at=$7 WHERE id=$8`
	_, err = r.db.Exec(ctx, q, p.Title, p.Description, p.Status, p.StartDate,
		p.EndDate, p.DepartmentID, p.UpdatedAt, id)
	return p, err
}

func (r *Repository) UpdateProgress(ctx context.Context, id string, progress int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE projects SET progress=$1, updated_at=$2 WHERE id=$3`,
		progress, time.Now(), id,
	)
	return err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	return err
}

func (r *Repository) Assign(ctx context.Context, projectID, userID string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO project_assignments (project_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		projectID, userID,
	)
	return err
}

func (r *Repository) Unassign(ctx context.Context, projectID, userID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM project_assignments WHERE project_id=$1 AND user_id=$2`,
		projectID, userID,
	)
	return err
}

func (r *Repository) ListDueTomorrow(ctx context.Context) ([]*Project, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+selectCols+` FROM projects
		 WHERE end_date = CURRENT_DATE + 1
		 AND status NOT IN ('completed', 'cancelled')
		 ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var projects []*Project
	for rows.Next() {
		p := &Project{}
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Status, &p.Progress,
			&p.StartDate, &p.EndDate, &p.DepartmentID, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (r *Repository) GetAssignees(ctx context.Context, projectID string) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT user_id FROM project_assignments WHERE project_id=$1`, projectID,
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
