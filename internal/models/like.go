package models

type Like struct {
	ID        int64
	UserID    int64
	PostID    *int64
	CommentID *int64
	Type      int
}
