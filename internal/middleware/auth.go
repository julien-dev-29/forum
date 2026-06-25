package middleware

import (
	"context"
	"database/sql"
	"net/http"

	"forum/internal/session"
)

type contextKey string

const ContextKeyUserID contextKey = "user_id"

func Auth(db *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := session.ReadCookie(r)
		if token != "" {
			sess, err := session.GetByToken(db, token)
			if err == nil {
				ctx := context.WithValue(r.Context(), ContextKeyUserID, sess.UserID)
				r = r.WithContext(ctx)
			} else {
				session.DeleteCookie(w)
			}
		}
		next.ServeHTTP(w, r)
	})
}
