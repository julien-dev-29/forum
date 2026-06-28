package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"forum/internal/database/sqlite"
)

type adminHandler struct {
	db *sql.DB
}

func (h *adminHandler) dashboard(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		renderError(w, http.StatusForbidden)
		return
	}

	users, err := sqlite.GetAllUsers(h.db)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	reports, err := sqlite.GetPendingReports(h.db)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	categories, err := sqlite.GetAllCategories(h.db)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	modRequests, err := sqlite.GetAllModRequests(h.db)
	if err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "admin.html", map[string]any{
		"Authenticated": true,
		"Username":      getUsername(r),
		"Role":          getRole(r),
		"Users":         users,
		"Reports":       reports,
		"Categories":    categories,
		"ModRequests":   modRequests,
		"UnreadCount":   getUnreadCount(h.db, r),
		"CSRFToken":     getCSRFToken(w, r),
	})
}

func (h *adminHandler) promoteUser(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		renderError(w, http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	userIDStr := r.FormValue("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	if err := sqlite.UpdateUserRole(h.db, userID, "moderator"); err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *adminHandler) demoteUser(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		renderError(w, http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	userIDStr := r.FormValue("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	if err := sqlite.UpdateUserRole(h.db, userID, "user"); err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *adminHandler) respondReport(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		renderError(w, http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	reportIDStr := r.FormValue("report_id")
	reportID, err := strconv.ParseInt(reportIDStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	status := r.FormValue("status")
	if status != "reviewed" && status != "dismissed" {
		renderError(w, http.StatusBadRequest)
		return
	}

	adminResponse := strings.TrimSpace(r.FormValue("admin_response"))

	if err := sqlite.UpdateReportStatus(h.db, reportID, status, adminResponse); err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *adminHandler) createCategory(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		renderError(w, http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	if err := sqlite.CreateCategory(h.db, name); err != nil {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *adminHandler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		renderError(w, http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	catIDStr := r.FormValue("id")
	catID, err := strconv.ParseInt(catIDStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	if err := sqlite.DeleteCategory(h.db, catID); err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *adminHandler) approveModRequest(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		renderError(w, http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	reqIDStr := r.FormValue("request_id")
	reqID, err := strconv.ParseInt(reqIDStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	req, err := sqlite.GetModRequestByID(h.db, reqID)
	if err != nil {
		renderError(w, http.StatusNotFound)
		return
	}

	if err := sqlite.UpdateModRequestStatus(h.db, reqID, "approved"); err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	if err := sqlite.UpdateUserRole(h.db, req.UserID, "moderator"); err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *adminHandler) denyModRequest(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		renderError(w, http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	reqIDStr := r.FormValue("request_id")
	reqID, err := strconv.ParseInt(reqIDStr, 10, 64)
	if err != nil {
		renderError(w, http.StatusBadRequest)
		return
	}

	if err := sqlite.UpdateModRequestStatus(h.db, reqID, "denied"); err != nil {
		renderError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
