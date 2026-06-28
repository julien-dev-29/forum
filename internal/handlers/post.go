package handlers

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"forum/internal/database/sqlite"
	"forum/internal/models"

	"github.com/google/uuid"
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
		"CurrentUserID": getUserIDInt(r),
		"Username":      getUsername(r),
		"Role":          getRole(r),
		"Posts":         posts,
		"Categories":    categories,
		"SelectedCat":   categoryFilter,
		"CurrentFilter": getFilterType(myPosts, liked),
		"UnreadCount":   getUnreadCount(h.db, r),
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
		"Role":          getRole(r),
		"Categories":    categories,
		"UnreadCount":   getUnreadCount(h.db, r),
	})
}

func (h *postHandler) createPostPost(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 21<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
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
			"Role":          getRole(r),
			"Categories":    categories,
			"Error":         "Title, content, and at least one category are required",
			"UnreadCount":   getUnreadCount(h.db, r),
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

	imagePath, err := h.saveUploadedImage(r)
	if err != nil {
		categories, _ := sqlite.GetAllCategories(h.db)
		renderTemplate(w, "create-post.html", map[string]any{
			"Authenticated": true,
			"Username":      getUsername(r),
			"Role":          getRole(r),
			"Categories":    categories,
			"Error":         err.Error(),
			"UnreadCount":   getUnreadCount(h.db, r),
		})
		return
	}

	_, err = sqlite.CreatePost(h.db, *userID, title, content, imagePath, ids)
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
		"CurrentUserID": getUserIDInt(r),
		"Username":      getUsername(r),
		"Role":          getRole(r),
		"Post":          post,
		"Comments":      comments,
		"UnreadCount":   getUnreadCount(h.db, r),
	})
}

func (h *postHandler) editGet(w http.ResponseWriter, r *http.Request) {
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
	post, err := sqlite.GetPostByID(h.db, id, userID)
	if err != nil {
		renderError(w, http.StatusNotFound)
		return
	}

	if userID == nil || post.UserID != *userID {
		renderError(w, http.StatusForbidden)
		return
	}

	categories, err := sqlite.GetAllCategories(h.db)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	unreadCount, err := sqlite.GetUnreadNotificationCount(h.db, *userID)
	if err != nil {
		unreadCount = 0
	}

	selectedIDs := make(map[int64]bool)
	for _, c := range post.Categories {
		selectedIDs[c.ID] = true
	}

	renderTemplate(w, "edit-post.html", map[string]any{
		"Authenticated":  true,
		"UserID":         userID,
		"Username":       getUsername(r),
		"Role":           getRole(r),
		"Post":           post,
		"Categories":     categories,
		"SelectedCatIDs": selectedIDs,
		"UnreadCount":    unreadCount,
	})
}

func (h *postHandler) editPost(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 21<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	idStr := r.FormValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))
	categoryIDs := r.Form["categories"]

	if title == "" || content == "" || len(categoryIDs) == 0 {
		userID := getUserID(r)
		post, _ := sqlite.GetPostByID(h.db, id, userID)
		categories, _ := sqlite.GetAllCategories(h.db)
		selectedIDs := make(map[int64]bool)
		if post != nil {
			for _, c := range post.Categories {
				selectedIDs[c.ID] = true
			}
		}
		renderTemplate(w, "edit-post.html", map[string]any{
			"Authenticated":  true,
			"UserID":         userID,
			"Username":       getUsername(r),
			"Role":           getRole(r),
			"Post":           post,
			"Categories":     categories,
			"SelectedCatIDs": selectedIDs,
			"Error":          "Title, content, and at least one category are required",
			"UnreadCount":    getUnreadCount(h.db, r),
		})
		return
	}

	var ids []int64
	for _, cid := range categoryIDs {
		cidInt, err := strconv.ParseInt(cid, 10, 64)
		if err != nil {
			renderError(w, http.StatusBadRequest)
			return
		}
		ids = append(ids, cidInt)
	}

	userID := getUserID(r)
	if userID == nil {
		renderError(w, http.StatusUnauthorized)
		return
	}

	imagePath, err := h.handleEditImage(r, id, *userID)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	if err := sqlite.UpdatePost(h.db, id, *userID, title, content, imagePath, ids); err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/post?id="+idStr, http.StatusSeeOther)
}

func (h *postHandler) delete(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
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

	userID := getUserID(r)
	if userID == nil {
		renderError(w, http.StatusUnauthorized)
		return
	}

	imagePath, _ := sqlite.GetPostImagePath(h.db, postID, *userID)

	if err := sqlite.DeletePost(h.db, postID, *userID); err != nil {
		renderError(w, http.StatusForbidden)
		return
	}

	if imagePath != "" {
		filePath := filepath.Join("ui", "static", "uploads", filepath.Base(imagePath))
		os.Remove(filePath)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

var allowedExtensions = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
}

var magicBytes = map[string][]byte{
	"image/jpeg": {0xff, 0xd8, 0xff},
	"image/png":  {0x89, 0x50, 0x4e, 0x47},
	"image/gif":  {0x47, 0x49, 0x46, 0x38},
}

func uploadsDir() string {
	return filepath.Join("ui", "static", "uploads")
}

func (h *postHandler) saveUploadedImage(r *http.Request) (string, error) {
	file, header, err := r.FormFile("image")
	if err != nil {
		return "", nil
	}
	defer file.Close()

	if header.Size > 20<<20 {
		return "", fmt.Errorf("image too large: maximum size is 20 MB")
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if _, ok := allowedExtensions[ext]; !ok {
		return "", fmt.Errorf("unsupported image type: only JPEG, PNG, and GIF are allowed")
	}

	buf := make([]byte, 512)
	if _, err := io.ReadFull(file, buf); err != nil {
		return "", fmt.Errorf("read image: %w", err)
	}

	contentType := allowedExtensions[ext]
	magic, ok := magicBytes[contentType]
	if !ok || !bytes.HasPrefix(buf, magic) {
		return "", fmt.Errorf("invalid image file")
	}

	file.Seek(0, io.SeekStart)

	filename := uuid.New().String() + ext
	destPath := filepath.Join(uploadsDir(), filename)
	dst, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("save image: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("write image: %w", err)
	}

	return "/static/uploads/" + filename, nil
}

func (h *postHandler) handleEditImage(r *http.Request, postID, userID int64) (string, error) {
	removeImage := r.FormValue("remove_image") == "1"

	newPath, err := h.saveUploadedImage(r)
	if err != nil {
		return "", err
	}

	if newPath != "" {
		oldPath, _ := sqlite.GetPostImagePath(h.db, postID, userID)
		if oldPath != "" {
			os.Remove(filepath.Join("ui", "static", "uploads", filepath.Base(oldPath)))
		}
		return newPath, nil
	}

	if removeImage {
		oldPath, _ := sqlite.GetPostImagePath(h.db, postID, userID)
		if oldPath != "" {
			os.Remove(filepath.Join("ui", "static", "uploads", filepath.Base(oldPath)))
		}
		return "", nil
	}

	existingPath, _ := sqlite.GetPostImagePath(h.db, postID, userID)
	return existingPath, nil
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
