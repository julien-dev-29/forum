package models

import "time"

type Notification struct {
	ID             int64
	UserID         int64
	ActorID        int64
	ActorName      string
	Type           string
	PostID         int64
	PostTitle      string
	CommentID      *int64
	CommentContent string
	IsRead         bool
	CreatedAt      time.Time
}
