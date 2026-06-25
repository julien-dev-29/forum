package models

import "time"

type Comment struct {
	ID           int64
	PostID       int64
	UserID       int64
	Content      string
	CreatedAt    time.Time
	Username     string
	LikeCount    int
	DislikeCount int
	UserLike     int
}
