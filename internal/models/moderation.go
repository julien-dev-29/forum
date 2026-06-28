package models

import "time"

type ModRequest struct {
	ID        int64
	UserID    int64
	Username  string
	Status    string
	CreatedAt time.Time
}
