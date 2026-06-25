package handlers

import (
	"database/sql"
	"net/http"

	"forum/internal/database/sqlite"
)

type notificationHandler struct {
	db *sql.DB
}

func (h *notificationHandler) list(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID := getUserID(r)
	if userID == nil {
		renderError(w, http.StatusUnauthorized)
		return
	}

	notifs, err := sqlite.GetNotificationsByUser(h.db, *userID)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	unreadCount, err := sqlite.GetUnreadNotificationCount(h.db, *userID)
	if err != nil {
		unreadCount = 0
	}

	renderTemplate(w, "notifications.html", map[string]any{
		"Authenticated": true,
		"UserID":        userID,
		"Username":      getUsername(r),
		"Notifications": notifs,
		"UnreadCount":   unreadCount,
	})
}

func (h *notificationHandler) markRead(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID := getUserID(r)
	if userID == nil {
		renderError(w, http.StatusUnauthorized)
		return
	}

	_ = sqlite.MarkNotificationsRead(h.db, *userID)
	http.Redirect(w, r, "/notifications", http.StatusSeeOther)
}
