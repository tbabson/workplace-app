package projectcomment

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const cols = `id, project_id, parent_id, author_id, content, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (*Comment, error) {
	c := &Comment{}
	return c, row.Scan(&c.ID, &c.ProjectID, &c.ParentID, &c.AuthorID, &c.Content, &c.CreatedAt, &c.UpdatedAt)
}

func (r *Repository) Create(ctx context.Context, c *Comment) error {
	q := `INSERT INTO project_comments (project_id, parent_id, author_id, content)
	      VALUES ($1,$2,$3,$4) RETURNING ` + cols
	got, err := scan(r.db.QueryRow(ctx, q, c.ProjectID, c.ParentID, c.AuthorID, c.Content))
	if err != nil {
		return err
	}
	*c = *got
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Comment, error) {
	return scan(r.db.QueryRow(ctx, `SELECT `+cols+` FROM project_comments WHERE id=$1`, id))
}

func (r *Repository) ListByProject(ctx context.Context, projectID string) ([]*Comment, error) {
	rows, err := r.db.Query(ctx, `SELECT `+cols+` FROM project_comments WHERE project_id=$1 ORDER BY created_at ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Comment
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

func (r *Repository) Update(ctx context.Context, id, content string) (*Comment, error) {
	c, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	c.Content = content
	c.UpdatedAt = time.Now()
	_, err = r.db.Exec(ctx, `UPDATE project_comments SET content=$1, updated_at=$2 WHERE id=$3`, content, c.UpdatedAt, id)
	return c, err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM project_comments WHERE id=$1`, id)
	return err
}
