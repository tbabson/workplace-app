package project

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type File struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	Size       int64     `json:"size"`
	UploadedBy string    `json:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at"`
}

type AddFileDTO struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Size int64  `json:"size"`
}

type FileRepository struct{ db *pgxpool.Pool }

func NewFileRepository(db *pgxpool.Pool) *FileRepository { return &FileRepository{db: db} }

func (r *FileRepository) Add(ctx context.Context, f *File) error {
	q := `INSERT INTO project_files (project_id, name, url, size, uploaded_by)
	      VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at`
	return r.db.QueryRow(ctx, q, f.ProjectID, f.Name, f.URL, f.Size, f.UploadedBy).
		Scan(&f.ID, &f.CreatedAt)
}

func (r *FileRepository) List(ctx context.Context, projectID string) ([]*File, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, name, url, size, uploaded_by, created_at
		 FROM project_files WHERE project_id=$1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*File
	for rows.Next() {
		f := &File{}
		if err := rows.Scan(&f.ID, &f.ProjectID, &f.Name, &f.URL, &f.Size, &f.UploadedBy, &f.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, nil
}

func (r *FileRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM project_files WHERE id=$1`, id)
	return err
}
