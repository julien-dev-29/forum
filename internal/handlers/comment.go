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

func (h *commentHandler) editGet(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	userID := getUserID(r)
	if userID == nil {
		renderError(w, http.StatusUnauthorized)
		return
	}

	comment, err := sqlite.GetCommentByID(h.db, id)
	if err != nil {
		renderError(w, http.StatusNotFound)
		return
	}

	if comment.UserID != *userID {
		renderError(w, http.StatusForbidden)
		return
	}

	post, err := sqlite.GetPostByID(h.db, comment.PostID, userID)
	if err != nil {
		renderError(w, http.StatusNotFound)
		return
	}

	renderTemplate(w, "edit-comment.html", map[string]any{
		"Authenticated": true,
		"UserID":        userID,
		"Username":      getUsername(r),
		"Role":          getRole(r),
		"Comment":       comment,
		"Post":          post,
		"UnreadCount":   getUnreadCount(h.db, r),
		"CSRFToken":     getCSRFToken(w, r),
	})
}

func (h *commentHandler) editPost(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	commentIDStr := r.FormValue("id")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		renderError(w, http.StatusBadRequest)
		return
	}

	userID := getUserID(r)
	if userID == nil {
		renderError(w, http.StatusUnauthorized)
		return
	}

	if err := sqlite.UpdateComment(h.db, commentID, *userID, content); err != nil {
		renderError(w, http.StatusForbidden)
		return
	}

	comment, err := sqlite.GetCommentByID(h.db, commentID)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/post?id="+strconv.FormatInt(comment.PostID, 10), http.StatusSeeOther)
}

func (h *commentHandler) delete(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	commentIDStr := r.FormValue("id")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	comment, err := sqlite.GetCommentByID(h.db, commentID)
	if err != nil {
		renderError(w, http.StatusNotFound)
		return
	}

	userID := getUserID(r)
	if userID == nil {
		renderError(w, http.StatusUnauthorized)
		return
	}

	if err := sqlite.DeleteComment(h.db, commentID, *userID); err != nil {
		renderError(w, http.StatusForbidden)
		return
	}

	http.Redirect(w, r, "/post?id="+strconv.FormatInt(comment.PostID, 10), http.StatusSeeOther)
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
