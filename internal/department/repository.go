package department

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

const cols = `id, name, parent_id, head_id, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (*Department, error) {
	d := &Department{}
	return d, row.Scan(&d.ID, &d.Name, &d.ParentID, &d.HeadID, &d.CreatedAt, &d.UpdatedAt)
}

func (r *Repository) Create(ctx context.Context, d *Department) error {
	q := `INSERT INTO departments (name, parent_id, head_id)
	      VALUES ($1, $2, $3)
	      RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, q, d.Name, d.ParentID, d.HeadID).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Department, error) {
	return scan(r.db.QueryRow(ctx, `SELECT `+cols+` FROM departments WHERE id = $1`, id))
}

func (r *Repository) List(ctx context.Context) ([]*Department, error) {
	rows, err := r.db.Query(ctx, `SELECT `+cols+` FROM departments ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var depts []*Department
	for rows.Next() {
		d, err := scan(rows)
		if err != nil {
			return nil, err
		}
		depts = append(depts, d)
	}
	return depts, rows.Err()
}

// HasChildren reports whether any department sits under id.
func (r *Repository) HasChildren(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM departments WHERE parent_id = $1)`, id).Scan(&ok)
	return ok, err
}

func (r *Repository) Update(ctx context.Context, id string, dto UpdateDepartmentDTO) (*Department, error) {
	d, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if dto.Name != nil {
		d.Name = *dto.Name
	}
	if dto.ParentID != nil {
		if *dto.ParentID == "" {
			d.ParentID = nil
		} else {
			d.ParentID = dto.ParentID
		}
	}
	if dto.HeadID != nil {
		if *dto.HeadID == "" {
			d.HeadID = nil // "" clears the head
		} else {
			d.HeadID = dto.HeadID
		}
	}
	d.UpdatedAt = time.Now()

	q := `UPDATE departments SET name=$1, parent_id=$2, head_id=$3, updated_at=$4 WHERE id=$5`
	_, err = r.db.Exec(ctx, q, d.Name, d.ParentID, d.HeadID, d.UpdatedAt, id)
	return d, err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM departments WHERE id = $1`, id)
	return err
}
