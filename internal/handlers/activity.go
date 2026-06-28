package handlers

import (
	"database/sql"
	"net/http"

	"forum/internal/database/sqlite"
)

type activityHandler struct {
	db *sql.DB
}

func (h *activityHandler) show(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID := getUserID(r)
	if userID == nil {
		renderError(w, http.StatusUnauthorized)
		return
	}

	posts, err := sqlite.GetPostsByUser(h.db, *userID, userID)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	comments, err := sqlite.GetUserCommentsWithPosts(h.db, *userID)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	likes, err := sqlite.GetUserLikesWithPosts(h.db, *userID)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	unreadCount, err := sqlite.GetUnreadNotificationCount(h.db, *userID)
	if err != nil {
		unreadCount = 0
	}

	renderTemplate(w, "activity.html", map[string]any{
		"Authenticated": true,
		"UserID":        userID,
		"Username":      getUsername(r),
		"Role":          getRole(r),
		"Posts":         posts,
		"Comments":      comments,
		"Likes":         likes,
		"UnreadCount":   unreadCount,
	})
}
