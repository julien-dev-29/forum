package models

import "time"

type Post struct {
	ID           int64
	UserID       int64
	Title        string
	Content      string
	CreatedAt    time.Time
	Username     string
	Categories   []Category
	LikeCount    int
	DislikeCount int
	UserLike     int
}
