package handlers

import (
	"database/sql"
	"net/http"

	"forum/internal/middleware"
)

type contextKey string

const (
	contextKeyUserID   = middleware.ContextKeyUserID
	contextKeyUsername = middleware.ContextKeyUsername
	contextKeyRole     = middleware.ContextKeyRole
)

type Handler struct {
	Auth        *authHandler
	OAuth       *oauthHandler
	Post        *postHandler
	Comment     *commentHandler
	Like        *likeHandler
	Notif       *notificationHandler
	Activity    *activityHandler
	Admin       *adminHandler
	Moderator   *moderatorHandler
	ModRequest  *modRequestHandler
}

func New(db *sql.DB) *Handler {
	return &Handler{
		Auth:        &authHandler{db: db},
		OAuth:       &oauthHandler{db: db},
		Post:        &postHandler{db: db},
		Comment:     &commentHandler{db: db},
		Like:        &likeHandler{db: db},
		Notif:       &notificationHandler{db: db},
		Activity:    &activityHandler{db: db},
		Admin:       &adminHandler{db: db},
		Moderator:   &moderatorHandler{db: db},
		ModRequest:  &modRequestHandler{db: db},
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

func (h *Handler) LoginGoogle(w http.ResponseWriter, r *http.Request) {
	h.OAuth.loginGoogle(w, r)
}

func (h *Handler) CallbackGoogle(w http.ResponseWriter, r *http.Request) {
	h.OAuth.callbackGoogle(w, r)
}

func (h *Handler) LoginGitHub(w http.ResponseWriter, r *http.Request) {
	h.OAuth.loginGitHub(w, r)
}

func (h *Handler) CallbackGitHub(w http.ResponseWriter, r *http.Request) {
	h.OAuth.callbackGitHub(w, r)
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

func (h *Handler) NotifList(w http.ResponseWriter, r *http.Request) {
	h.Notif.list(w, r)
}

func (h *Handler) NotifRead(w http.ResponseWriter, r *http.Request) {
	h.Notif.markRead(w, r)
}

func (h *Handler) NotifStream(w http.ResponseWriter, r *http.Request) {
	h.Notif.stream(w, r)
}

func (h *Handler) ActivityShow(w http.ResponseWriter, r *http.Request) {
	h.Activity.show(w, r)
}

func (h *Handler) EditPostGet(w http.ResponseWriter, r *http.Request) {
	h.Post.editGet(w, r)
}

func (h *Handler) EditPostPost(w http.ResponseWriter, r *http.Request) {
	h.Post.editPost(w, r)
}

func (h *Handler) DeletePost(w http.ResponseWriter, r *http.Request) {
	h.Post.delete(w, r)
}

func (h *Handler) EditCommentGet(w http.ResponseWriter, r *http.Request) {
	h.Comment.editGet(w, r)
}

func (h *Handler) EditCommentPost(w http.ResponseWriter, r *http.Request) {
	h.Comment.editPost(w, r)
}

func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	h.Comment.delete(w, r)
}

// Admin
func (h *Handler) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	h.Admin.dashboard(w, r)
}

func (h *Handler) AdminPromoteUser(w http.ResponseWriter, r *http.Request) {
	h.Admin.promoteUser(w, r)
}

func (h *Handler) AdminDemoteUser(w http.ResponseWriter, r *http.Request) {
	h.Admin.demoteUser(w, r)
}

func (h *Handler) AdminRespondReport(w http.ResponseWriter, r *http.Request) {
	h.Admin.respondReport(w, r)
}

func (h *Handler) AdminCreateCategory(w http.ResponseWriter, r *http.Request) {
	h.Admin.createCategory(w, r)
}

func (h *Handler) AdminDeleteCategory(w http.ResponseWriter, r *http.Request) {
	h.Admin.deleteCategory(w, r)
}

func (h *Handler) AdminApproveModRequest(w http.ResponseWriter, r *http.Request) {
	h.Admin.approveModRequest(w, r)
}

func (h *Handler) AdminDenyModRequest(w http.ResponseWriter, r *http.Request) {
	h.Admin.denyModRequest(w, r)
}

// Moderator
func (h *Handler) ModReportPost(w http.ResponseWriter, r *http.Request) {
	h.Moderator.reportPost(w, r)
}

func (h *Handler) ModReportComment(w http.ResponseWriter, r *http.Request) {
	h.Moderator.reportComment(w, r)
}

func (h *Handler) ModDeletePost(w http.ResponseWriter, r *http.Request) {
	h.Moderator.deletePost(w, r)
}

func (h *Handler) ModDeleteComment(w http.ResponseWriter, r *http.Request) {
	h.Moderator.deleteComment(w, r)
}

// ModRequest
func (h *Handler) ModRequestGet(w http.ResponseWriter, r *http.Request) {
	h.ModRequest.get(w, r)
}

func (h *Handler) ModRequestPost(w http.ResponseWriter, r *http.Request) {
	h.ModRequest.post(w, r)
}
