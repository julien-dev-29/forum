package handlers

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"forum/internal/database/sqlite"
)

type moderatorHandler struct {
	db *sql.DB
}

func (h *moderatorHandler) reportPost(w http.ResponseWriter, r *http.Request) {
	if !isModOrAdmin(r) {
		renderError(w, http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	postIDStr := r.FormValue("post_id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	reason := r.FormValue("reason")
	if reason != "irrelevant" && reason != "obscene" && reason != "illegal" && reason != "insulting" {
		renderError(w, http.StatusBadRequest)
		return
	}

	customText := strings.TrimSpace(r.FormValue("custom_text"))
	reporterID := getUserIDInt(r)

	if err := sqlite.CreateReport(h.db, reporterID, &postID, nil, reason, customText); err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/post?id="+postIDStr, http.StatusSeeOther)
}

func (h *moderatorHandler) reportComment(w http.ResponseWriter, r *http.Request) {
	if !isModOrAdmin(r) {
		renderError(w, http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	commentIDStr := r.FormValue("comment_id")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	postIDStr := r.FormValue("post_id")
	reason := r.FormValue("reason")
	if reason != "irrelevant" && reason != "obscene" && reason != "illegal" && reason != "insulting" {
		renderError(w, http.StatusBadRequest)
		return
	}

	customText := strings.TrimSpace(r.FormValue("custom_text"))
	reporterID := getUserIDInt(r)

	if err := sqlite.CreateReport(h.db, reporterID, nil, &commentID, reason, customText); err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/post?id="+postIDStr, http.StatusSeeOther)
}

func (h *moderatorHandler) deletePost(w http.ResponseWriter, r *http.Request) {
	if !isModOrAdmin(r) {
		renderError(w, http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	postIDStr := r.FormValue("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	imagePath, _ := sqlite.GetPostImagePathByID(h.db, postID)

	if err := sqlite.DeletePostByID(h.db, postID); err != nil {
		renderError(w, http.StatusNotFound)
		return
	}

	if imagePath != "" {
		filePath := filepath.Join("ui", "static", "uploads", filepath.Base(imagePath))
		os.Remove(filePath)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *moderatorHandler) deleteComment(w http.ResponseWriter, r *http.Request) {
	if !isModOrAdmin(r) {
		renderError(w, http.StatusForbidden)
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

	postIDStr := r.FormValue("post_id")

	comment, err := sqlite.GetCommentByID(h.db, commentID)
	if err != nil {
		renderError(w, http.StatusNotFound)
		return
	}

	if err := sqlite.DeleteCommentByID(h.db, commentID); err != nil {
		renderError(w, http.StatusNotFound)
		return
	}

	redirectTo := "/post?id=" + postIDStr
	if postIDStr == "" {
		redirectTo = "/post?id=" + strconv.FormatInt(comment.PostID, 10)
	}
	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}
