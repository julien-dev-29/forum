package session

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        int64
	UserID    int64
	Token     string
	ExpiresAt time.Time
}

const cookieName = "session_token"
const sessionDuration = 24 * time.Hour

func Create(db *sql.DB, userID int64) (string, error) {
	if err := deleteUserSessions(db, userID); err != nil {
		return "", fmt.Errorf("delete old sessions: %w", err)
	}

	token := uuid.New().String()
	expiresAt := time.Now().Add(sessionDuration)

	_, err := db.Exec(
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		userID, token, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	return token, nil
}

func GetByToken(db *sql.DB, token string) (*Session, error) {
	s := &Session{}
	err := db.QueryRow(
		"SELECT id, user_id, token, expires_at FROM sessions WHERE token = ?",
		token,
	).Scan(&s.ID, &s.UserID, &s.Token, &s.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if time.Now().After(s.ExpiresAt) {
		Delete(db, token)
		return nil, fmt.Errorf("session expired")
	}
	return s, nil
}

func Delete(db *sql.DB, token string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func deleteUserSessions(db *sql.DB, userID int64) error {
	_, err := db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

func CleanupExpired(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now())
	return err
}

func WriteCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(sessionDuration),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func DeleteCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func ReadCookie(r *http.Request) string {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return c.Value
}
