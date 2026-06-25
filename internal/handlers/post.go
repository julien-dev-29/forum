package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"forum/internal/database/sqlite"
	"forum/internal/models"
)

type postHandler struct {
	db *sql.DB
}

func (h *postHandler) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		renderError(w, http.StatusNotFound)
		return
	}

	userID := getUserID(r)
	categoryFilter := r.URL.Query().Get("category")
	myPosts := r.URL.Query().Get("my-posts")
	liked := r.URL.Query().Get("liked")

	categories, err := sqlite.GetAllCategories(h.db)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	var posts []models.Post

	if myPosts == "1" && userID != nil {
		posts, err = sqlite.GetPostsByUser(h.db, *userID, userID)
	} else if liked == "1" && userID != nil {
		posts, err = sqlite.GetLikedPosts(h.db, *userID)
	} else if categoryFilter != "" {
		catID, parseErr := strconv.ParseInt(categoryFilter, 10, 64)
		if parseErr != nil {
			renderError(w, http.StatusBadRequest)
			return
		}
		posts, err = sqlite.GetPostsByCategory(h.db, catID, userID)
	} else {
		posts, err = sqlite.GetAllPosts(h.db, userID)
	}

	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "index.html", map[string]any{
		"Authenticated": isAuthenticated(r),
		"UserID":        userID,
		"Username":      getUsername(r),
		"Posts":         posts,
		"Categories":    categories,
		"SelectedCat":   categoryFilter,
		"CurrentFilter": getFilterType(myPosts, liked),
	})
}

func (h *postHandler) createPostGet(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	categories, err := sqlite.GetAllCategories(h.db)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "create-post.html", map[string]any{
		"Authenticated": true,
		"Username":      getUsername(r),
		"Categories":    categories,
	})
}

func (h *postHandler) createPostPost(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))
	categoryIDs := r.Form["categories"]

	if title == "" || content == "" || len(categoryIDs) == 0 {
		categories, _ := sqlite.GetAllCategories(h.db)
		renderTemplate(w, "create-post.html", map[string]any{
			"Authenticated": true,
			"Username":      getUsername(r),
			"Categories":    categories,
			"Error":         "Title, content, and at least one category are required",
		})
		return
	}

	var ids []int64
	for _, cid := range categoryIDs {
		id, err := strconv.ParseInt(cid, 10, 64)
		if err != nil {
			renderError(w, http.StatusBadRequest)
			return
		}
		ids = append(ids, id)
	}

	userID := getUserID(r)
	if userID == nil {
		renderError(w, http.StatusUnauthorized)
		return
	}

	_, err := sqlite.CreatePost(h.db, *userID, title, content, ids)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *postHandler) viewPost(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	userID := getUserID(r)

	post, err := sqlite.GetPostByID(h.db, id, userID)
	if err != nil {
		renderError(w, http.StatusNotFound)
		return
	}

	comments, err := sqlite.GetCommentsByPostID(h.db, id, userID)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "view-post.html", map[string]any{
		"Authenticated": isAuthenticated(r),
		"UserID":        userID,
		"Username":      getUsername(r),
		"Post":          post,
		"Comments":      comments,
	})
}

func getFilterType(myPosts, liked string) string {
	if myPosts == "1" {
		return "my-posts"
	}
	if liked == "1" {
		return "liked"
	}
	return "all"
}
