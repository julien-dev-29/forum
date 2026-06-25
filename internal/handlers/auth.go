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
			"Error":         "All fields are required",
		})
		return
	}

	exists, err := sqlite.EmailExists(h.db, email)
	if err != nil || exists {
		renderTemplate(w, "register.html", map[string]any{
			"Authenticated": false,
			"Error":         "Email already taken",
		})
		return
	}

	if err := sqlite.CreateUser(h.db, email, username, password); err != nil {
		renderTemplate(w, "register.html", map[string]any{
			"Authenticated": false,
			"Error":         "Registration failed",
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
			"Error":         "Invalid email or password",
		})
		return
	}

	token, err := session.Create(h.db, user.ID)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	session.WriteCookie(w, token)
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
