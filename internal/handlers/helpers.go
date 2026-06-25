package handlers

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"
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
		"Status":  status,
		"Message": http.StatusText(status),
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
