package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

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
		"Role":          getRole(r),
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

func (h *notificationHandler) stream(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	lastCount := -1
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			count, err := sqlite.GetUnreadNotificationCount(h.db, *userID)
			if err != nil {
				continue
			}
			if count != lastCount {
				lastCount = count
				fmt.Fprintf(w, "data: {\"count\": %d}\n\n", count)
				flusher.Flush()
			}
		}
	}
}
