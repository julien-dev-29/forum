package csrf

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
)

const cookieName = "csrf_token"
const headerName = "X-CSRF-Token"
const formField = "csrf_token"

var Secure bool

func Configure(secure bool) {
	Secure = secure
}

func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   Secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	})
}

func Validate(r *http.Request) bool {
	token := tokenFromRequest(r)
	cookie := cookieFromRequest(r)
	if token == "" || cookie == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(cookie)) == 1
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST", "PUT", "PATCH", "DELETE":
			if !Validate(r) {
				http.Error(w, "403 Forbidden - invalid CSRF token", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func tokenFromRequest(r *http.Request) string {
	if t := r.Header.Get(headerName); t != "" {
		return t
	}
	if err := r.ParseForm(); err == nil {
		return r.FormValue(formField)
	}
	return ""
}

func cookieFromRequest(r *http.Request) string {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

func GetToken(r *http.Request) string {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return c.Value
}
