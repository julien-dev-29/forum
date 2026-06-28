package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"forum/internal/database/sqlite"
	"forum/internal/session"
)

type authHandler struct {
	db *sql.DB
}

func (h *authHandler) registerGet(w http.ResponseWriter, r *http.Request) {
	if isAuthenticated(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	renderTemplate(w, "register.html", map[string]any{
		"Authenticated": false,
		"Role":          "guest",
		"CSRFToken":     getCSRFToken(w, r),
	})
}

func (h *authHandler) registerPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if email == "" || username == "" || password == "" {
		renderTemplate(w, "register.html", map[string]any{
			"Authenticated": false,
			"Role":          "guest",
			"Error":         "All fields are required",
			"CSRFToken":     getCSRFToken(w, r),
		})
		return
	}

	exists, err := sqlite.EmailExists(h.db, email)
	if err != nil || exists {
		renderTemplate(w, "register.html", map[string]any{
			"Authenticated": false,
			"Role":          "guest",
			"Error":         "Email already taken",
			"CSRFToken":     getCSRFToken(w, r),
		})
		return
	}

	if err := sqlite.CreateUser(h.db, email, username, password); err != nil {
		renderTemplate(w, "register.html", map[string]any{
			"Authenticated": false,
			"Role":          "guest",
			"Error":         "Registration failed",
			"CSRFToken":     getCSRFToken(w, r),
		})
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *authHandler) loginGet(w http.ResponseWriter, r *http.Request) {
	if isAuthenticated(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	renderTemplate(w, "login.html", map[string]any{
		"Authenticated": false,
		"Role":          "guest",
		"CSRFToken":     getCSRFToken(w, r),
	})
}

func (h *authHandler) loginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	user, err := sqlite.AuthenticateUser(h.db, email, password)
	if err != nil {
		renderTemplate(w, "login.html", map[string]any{
			"Authenticated": false,
			"Role":          "guest",
			"Error":         "Invalid email or password",
			"CSRFToken":     getCSRFToken(w, r),
		})
		return
	}

	oldToken := session.ReadCookie(r)
	if oldToken != "" {
		newToken, err := session.Rotate(h.db, oldToken)
		if err == nil {
			session.DeleteCookie(w)
			session.WriteCookie(w, newToken)
		}
	} else {
		token, err := session.Create(h.db, user.ID)
		if err != nil {
			renderError(w, http.StatusInternalServerError)
			return
		}
		session.WriteCookie(w, token)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *authHandler) logoutPost(w http.ResponseWriter, r *http.Request) {
	token := session.ReadCookie(r)
	if token != "" {
		session.Delete(h.db, token)
	}
	session.DeleteCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
