package models

import "time"

type User struct {
	ID            int64
	Email         string
	Username      string
	Password      string
	OAuthProvider string
	OAuthID       string
	Role          string
	CreatedAt     time.Time
}
