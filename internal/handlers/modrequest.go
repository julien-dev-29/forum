package handlers

import (
	"database/sql"
	"net/http"

	"forum/internal/database/sqlite"
)

type modRequestHandler struct {
	db *sql.DB
}

func (h *modRequestHandler) get(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID := getUserIDInt(r)

	existing, err := sqlite.GetModRequestByUser(h.db, userID)
	if err != nil {
		renderTemplate(w, "request-mod.html", map[string]any{
			"Authenticated": true,
			"Username":      getUsername(r),
			"Role":          getRole(r),
			"Request":       nil,
			"UnreadCount":   getUnreadCount(h.db, r),
		})
		return
	}

	renderTemplate(w, "request-mod.html", map[string]any{
		"Authenticated": true,
		"Username":      getUsername(r),
		"Role":          getRole(r),
		"Request":       existing,
		"UnreadCount":   getUnreadCount(h.db, r),
	})
}

func (h *modRequestHandler) post(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID := getUserIDInt(r)

	_, err := sqlite.GetModRequestByUser(h.db, userID)
	if err == nil {
		http.Redirect(w, r, "/mod/request", http.StatusSeeOther)
		return
	}

	if err := sqlite.CreateModRequest(h.db, userID); err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/mod/request", http.StatusSeeOther)
}
