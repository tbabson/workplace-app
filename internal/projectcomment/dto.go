package projectcomment

type CreateCommentDTO struct {
	Content  string  `json:"content"`
	ParentID *string `json:"parent_id"`
}

type UpdateCommentDTO struct {
	Content string `json:"content"`
}
