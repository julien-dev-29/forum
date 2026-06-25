package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"forum/internal/database/sqlite"
)

type likeHandler struct {
	db *sql.DB
}

func (h *likeHandler) likePost(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	postIDStr := r.FormValue("post_id")
	likeTypeStr := r.FormValue("type")

	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	var likeType int
	switch likeTypeStr {
	case "like":
		likeType = 1
	case "dislike":
		likeType = -1
	default:
		renderError(w, http.StatusBadRequest)
		return
	}

	userID := getUserID(r)
	if userID == nil {
		renderError(w, http.StatusUnauthorized)
		return
	}

	if err := sqlite.TogglePostLike(h.db, *userID, postID, likeType); err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h *likeHandler) likeComment(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	commentIDStr := r.FormValue("comment_id")
	likeTypeStr := r.FormValue("type")

	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	var likeType int
	switch likeTypeStr {
	case "like":
		likeType = 1
	case "dislike":
		likeType = -1
	default:
		renderError(w, http.StatusBadRequest)
		return
	}

	userID := getUserID(r)
	if userID == nil {
		renderError(w, http.StatusUnauthorized)
		return
	}

	if err := sqlite.ToggleCommentLike(h.db, *userID, commentID, likeType); err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
