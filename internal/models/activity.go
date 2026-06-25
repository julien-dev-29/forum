package models

import "time"

type UserComment struct {
	CommentID int64
	Content   string
	CreatedAt time.Time
	PostID    int64
	PostTitle string
}

type UserLike struct {
	LikeType  int
	PostID    int64
	PostTitle string
	CreatedAt time.Time
	CommentID *int64
}
