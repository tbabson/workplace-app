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

func (r *Repository) Create(ctx context.Context, d *Department) error {
	q := `INSERT INTO departments (name, head_id)
	      VALUES ($1, $2)
	      RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, q, d.Name, d.HeadID).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Department, error) {
	q := `SELECT id, name, head_id, created_at, updated_at FROM departments WHERE id = $1`
	d := &Department{}
	err := r.db.QueryRow(ctx, q, id).Scan(&d.ID, &d.Name, &d.HeadID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *Repository) List(ctx context.Context) ([]*Department, error) {
	q := `SELECT id, name, head_id, created_at, updated_at FROM departments ORDER BY name`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var depts []*Department
	for rows.Next() {
		d := &Department{}
		if err := rows.Scan(&d.ID, &d.Name, &d.HeadID, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		depts = append(depts, d)
	}
	return depts, nil
}

func (r *Repository) Update(ctx context.Context, id string, dto UpdateDepartmentDTO) (*Department, error) {
	d, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if dto.Name != nil {
		d.Name = *dto.Name
	}
	if dto.HeadID != nil {
		d.HeadID = dto.HeadID
	}
	d.UpdatedAt = time.Now()

	q := `UPDATE departments SET name=$1, head_id=$2, updated_at=$3 WHERE id=$4`
	_, err = r.db.Exec(ctx, q, d.Name, d.HeadID, d.UpdatedAt, id)
	return d, err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM departments WHERE id = $1`, id)
	return err
}
