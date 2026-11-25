package project

import (
	"context"
	"time"

	"workplace/pkg/clock"

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

// List returns every project when scopeUserID is empty. Otherwise it returns the
// projects in deptID (and its sub-departments when withSubDepts is set) plus any
// project scopeUserID is assigned to.
func (r *Repository) List(ctx context.Context, deptID, scopeUserID string, withSubDepts bool) ([]*Project, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if scopeUserID != "" {
		rows, err = r.db.Query(ctx,
			`SELECT `+selectCols+` FROM projects
			 WHERE ($1 <> '' AND department_id IN (
			         SELECT id FROM departments WHERE id::text = $1 OR ($3 AND parent_id::text = $1)))
			    OR id IN (SELECT project_id FROM project_assignments WHERE user_id = $2)
			 ORDER BY created_at DESC`, deptID, scopeUserID, withSubDepts)
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
		 WHERE end_date = $1::date
		 AND status NOT IN ('completed', 'cancelled')
		 ORDER BY created_at DESC`, clock.DaysFromToday(1))
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

// IsAssigned reports whether userID is assigned to projectID.
func (r *Repository) IsAssigned(ctx context.Context, projectID, userID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM project_assignments WHERE project_id=$1 AND user_id=$2)`,
		projectID, userID).Scan(&ok)
	return ok, err
}

// ProjectsLedByDepartment returns the active projects (not completed or
// cancelled) that a dept_head overseeing deptID is assigned to: the head of
// deptID itself or of its parent department. exceptUserID is ignored as a head.
func (r *Repository) ProjectsLedByDepartment(ctx context.Context, deptID, exceptUserID string) ([]*Project, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+selectCols+` FROM projects
		 WHERE status NOT IN ('completed', 'cancelled')
		   AND id IN (
		     SELECT pa.project_id FROM project_assignments pa
		     JOIN users u ON u.id = pa.user_id
		     WHERE u.role = 'dept_head' AND u.id::text <> $2
		       AND u.department_id IN (
		         SELECT id FROM departments WHERE id::text = $1
		         UNION
		         SELECT parent_id FROM departments WHERE id::text = $1 AND parent_id IS NOT NULL))
		 ORDER BY created_at DESC`, deptID, exceptUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Project
	for rows.Next() {
		p := &Project{}
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Status, &p.Progress,
			&p.StartDate, &p.EndDate, &p.DepartmentID, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// UnassignOpenTasks clears userID from their unfinished tasks in projectID so the
// duties can be reassigned. Finished tasks keep their assignee as a record.
func (r *Repository) UnassignOpenTasks(ctx context.Context, projectID, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE project_tasks SET assigned_to = NULL, updated_at = $3
		 WHERE project_id = $1 AND assigned_to = $2 AND status <> 'done'`,
		projectID, userID, time.Now())
	return err
}

// ParentDepartment returns deptID's parent department ID, or "" if it is top-level.
func (r *Repository) ParentDepartment(ctx context.Context, deptID string) (string, error) {
	var parent *string
	err := r.db.QueryRow(ctx, `SELECT parent_id::text FROM departments WHERE id::text = $1`, deptID).Scan(&parent)
	if err != nil || parent == nil {
		return "", err
	}
	return *parent, nil
}

// ActiveProjectsInDepartment returns the projects of deptID that are not
// completed or cancelled.
func (r *Repository) ActiveProjectsInDepartment(ctx context.Context, deptID string) ([]*Project, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+selectCols+` FROM projects
		 WHERE department_id::text = $1 AND status NOT IN ('completed', 'cancelled')
		 ORDER BY created_at DESC`, deptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Project
	for rows.Next() {
		p := &Project{}
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Status, &p.Progress,
			&p.StartDate, &p.EndDate, &p.DepartmentID, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}
