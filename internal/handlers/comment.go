package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"forum/internal/database/sqlite"
)

type commentHandler struct {
	db *sql.DB
}

func (h *commentHandler) createComment(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	postIDStr := r.FormValue("post_id")
	content := strings.TrimSpace(r.FormValue("content"))

	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil || content == "" {
		renderError(w, http.StatusBadRequest)
		return
	}

	userID := getUserID(r)
	if userID == nil {
		renderError(w, http.StatusUnauthorized)
		return
	}

	if err := sqlite.CreateComment(h.db, postID, *userID, content); err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/post?id="+postIDStr, http.StatusSeeOther)
}
