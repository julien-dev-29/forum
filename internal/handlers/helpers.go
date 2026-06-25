package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"forum/internal/database/sqlite"
)

var templates *template.Template

func InitTemplates() {
	funcMap := template.FuncMap{
		"nowYear": func() int {
			return time.Now().Year()
		},
	}
	var err error
	templates, err = template.New("").Funcs(funcMap).ParseGlob(filepath.Join("ui", "html", "*.html"))
	if err != nil {
		log.Fatalf("parse templates: %v", err)
	}
}

func renderTemplate(w http.ResponseWriter, name string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render template %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func renderError(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
	renderTemplate(w, "error.html", map[string]any{
		"Status":        status,
		"Message":       http.StatusText(status),
		"Authenticated": false,
	})
}

func isAuthenticated(r *http.Request) bool {
	return r.Context().Value(contextKeyUserID) != nil
}

func getUserID(r *http.Request) *int64 {
	id, ok := r.Context().Value(contextKeyUserID).(int64)
	if !ok {
		return nil
	}
	return &id
}

func getUsername(r *http.Request) string {
	username, ok := r.Context().Value(contextKeyUsername).(string)
	if !ok {
		return ""
	}
	return username
}

func getUserIDInt(r *http.Request) int64 {
	id := getUserID(r)
	if id == nil {
		return 0
	}
	return *id
}

func getUnreadCount(db *sql.DB, r *http.Request) int {
	userID := getUserID(r)
	if userID == nil {
		return 0
	}
	count, err := sqlite.GetUnreadNotificationCount(db, *userID)
	if err != nil {
		return 0
	}
	return count
}
