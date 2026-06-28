package sqlite

import (
	"database/sql"
	"fmt"
	"forum/internal/models"
)

func CreateReport(db *sql.DB, reporterID int64, postID, commentID *int64, reason, customText string) error {
	_, err := db.Exec(
		"INSERT INTO reports (reporter_id, post_id, comment_id, reason, custom_text, status) VALUES (?, ?, ?, ?, ?, 'pending')",
		reporterID, postID, commentID, reason, customText,
	)
	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}
	return nil
}

func GetPendingReports(db *sql.DB) ([]models.Report, error) {
	query := `
		SELECT
			r.id, r.reporter_id, u.username,
			r.post_id, COALESCE(p.title, ''),
			r.comment_id, COALESCE(c.content, ''),
			r.reason, COALESCE(r.custom_text, ''),
			r.status, COALESCE(r.admin_response, ''),
			r.created_at
		FROM reports r
		JOIN users u ON u.id = r.reporter_id
		LEFT JOIN posts p ON p.id = r.post_id
		LEFT JOIN comments c ON c.id = r.comment_id
		ORDER BY r.created_at DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("get reports: %w", err)
	}
	defer rows.Close()

	var reports []models.Report
	for rows.Next() {
		var rep models.Report
		if err := rows.Scan(
			&rep.ID, &rep.ReporterID, &rep.ReporterName,
			&rep.PostID, &rep.PostTitle,
			&rep.CommentID, &rep.CommentContent,
			&rep.Reason, &rep.CustomText,
			&rep.Status, &rep.AdminResponse,
			&rep.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		reports = append(reports, rep)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	if reports == nil {
		reports = []models.Report{}
	}
	return reports, nil
}

func GetReportByID(db *sql.DB, id int64) (*models.Report, error) {
	rep := &models.Report{}
	var customText, adminResp *string
	err := db.QueryRow(`
		SELECT
			r.id, r.reporter_id, u.username,
			r.post_id, COALESCE(p.title, ''),
			r.comment_id, COALESCE(c.content, ''),
			r.reason, r.custom_text,
			r.status, r.admin_response,
			r.created_at
		FROM reports r
		JOIN users u ON u.id = r.reporter_id
		LEFT JOIN posts p ON p.id = r.post_id
		LEFT JOIN comments c ON c.id = r.comment_id
		WHERE r.id = ?
	`, id).Scan(
		&rep.ID, &rep.ReporterID, &rep.ReporterName,
		&rep.PostID, &rep.PostTitle,
		&rep.CommentID, &rep.CommentContent,
		&rep.Reason, &customText,
		&rep.Status, &adminResp,
		&rep.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get report by id: %w", err)
	}
	if customText != nil {
		rep.CustomText = *customText
	}
	if adminResp != nil {
		rep.AdminResponse = *adminResp
	}
	return rep, nil
}

func UpdateReportStatus(db *sql.DB, id int64, status, adminResponse string) error {
	_, err := db.Exec(
		"UPDATE reports SET status = ?, admin_response = ? WHERE id = ?",
		status, adminResponse, id,
	)
	if err != nil {
		return fmt.Errorf("update report status: %w", err)
	}
	return nil
}

func GetPostReportsCount(db *sql.DB, postID int64) (int, error) {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM reports WHERE post_id = ? AND status = 'pending'",
		postID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get post reports count: %w", err)
	}
	return count, nil
}
