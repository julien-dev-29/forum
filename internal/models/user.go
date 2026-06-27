package models

import "time"

type User struct {
	ID            int64
	Email         string
	Username      string
	Password      string
	OAuthProvider string
	OAuthID       string
	CreatedAt     time.Time
}
