package handlers

import (
	"database/sql"
	"net/http"

	"forum/internal/middleware"
)

type contextKey string

const contextKeyUserID = middleware.ContextKeyUserID

type Handler struct {
	Auth    *authHandler
	Post    *postHandler
	Comment *commentHandler
	Like    *likeHandler
}

func New(db *sql.DB) *Handler {
	return &Handler{
		Auth:    &authHandler{db: db},
		Post:    &postHandler{db: db},
		Comment: &commentHandler{db: db},
		Like:    &likeHandler{db: db},
	}
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	h.Post.home(w, r)
}

func (h *Handler) RegisterGet(w http.ResponseWriter, r *http.Request) {
	h.Auth.registerGet(w, r)
}

func (h *Handler) RegisterPost(w http.ResponseWriter, r *http.Request) {
	h.Auth.registerPost(w, r)
}

func (h *Handler) LoginGet(w http.ResponseWriter, r *http.Request) {
	h.Auth.loginGet(w, r)
}

func (h *Handler) LoginPost(w http.ResponseWriter, r *http.Request) {
	h.Auth.loginPost(w, r)
}

func (h *Handler) LogoutPost(w http.ResponseWriter, r *http.Request) {
	h.Auth.logoutPost(w, r)
}

func (h *Handler) CreatePostGet(w http.ResponseWriter, r *http.Request) {
	h.Post.createPostGet(w, r)
}

func (h *Handler) CreatePostPost(w http.ResponseWriter, r *http.Request) {
	h.Post.createPostPost(w, r)
}

func (h *Handler) ViewPost(w http.ResponseWriter, r *http.Request) {
	h.Post.viewPost(w, r)
}

func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	h.Comment.createComment(w, r)
}

func (h *Handler) LikePost(w http.ResponseWriter, r *http.Request) {
	h.Like.likePost(w, r)
}

func (h *Handler) LikeComment(w http.ResponseWriter, r *http.Request) {
	h.Like.likeComment(w, r)
}
