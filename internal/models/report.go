package models

import "time"

type Report struct {
	ID             int64
	ReporterID     int64
	ReporterName   string
	PostID         *int64
	PostTitle      string
	CommentID      *int64
	CommentContent string
	Reason         string
	CustomText     string
	Status         string
	AdminResponse  string
	CreatedAt      time.Time
}
