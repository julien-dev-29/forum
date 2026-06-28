package middleware

import (
	"context"
	"database/sql"
	"net/http"

	"forum/internal/database/sqlite"
	"forum/internal/session"
)

type contextKey string

const (
	ContextKeyUserID   contextKey = "user_id"
	ContextKeyUsername contextKey = "username"
	ContextKeyRole     contextKey = "role"
)

func Auth(db *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := session.ReadCookie(r)
		if token != "" {
			sess, err := session.GetByToken(db, token)
			if err == nil {
				ctx := context.WithValue(r.Context(), ContextKeyUserID, sess.UserID)
				user, err := sqlite.GetUserByID(db, sess.UserID)
				if err == nil {
					ctx = context.WithValue(ctx, ContextKeyUsername, user.Username)
					ctx = context.WithValue(ctx, ContextKeyRole, user.Role)
				}
				r = r.WithContext(ctx)
			} else {
				session.DeleteCookie(w)
			}
		}
		next.ServeHTTP(w, r)
	})
}
