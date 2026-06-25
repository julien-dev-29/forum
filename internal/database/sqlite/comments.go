package sqlite

import (
	"database/sql"
	"fmt"
	"forum/internal/models"
)

func CreateComment(db *sql.DB, postID, userID int64, content string) error {
	_, err := db.Exec(
		"INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)",
		postID, userID, content,
	)
	if err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	return nil
}

func GetCommentsByPostID(db *sql.DB, postID int64, userID *int64) ([]models.Comment, error) {
	rows, err := db.Query(`
		SELECT c.id, c.post_id, c.user_id, c.content, c.created_at, u.username
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.post_id = ?
		ORDER BY c.created_at ASC
	`, postID)
	if err != nil {
		return nil, fmt.Errorf("get comments: %w", err)
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var createdAt string
		com := models.Comment{}
		if err := rows.Scan(&com.ID, &com.PostID, &com.UserID, &com.Content, &createdAt, &com.Username); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		com.CreatedAt = parseTime(createdAt)

		likes, dislikes, err := getCommentLikeCounts(db, com.ID)
		if err != nil {
			return nil, err
		}
		com.LikeCount = likes
		com.DislikeCount = dislikes

		if userID != nil {
			like, err := getUserCommentLike(db, *userID, com.ID)
			if err != nil {
				return nil, err
			}
			com.UserLike = like
		}

		comments = append(comments, com)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	if comments == nil {
		comments = []models.Comment{}
	}
	return comments, nil
}

func getCommentLikeCounts(db *sql.DB, commentID int64) (likes, dislikes int, err error) {
	err = db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN type = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = -1 THEN 1 ELSE 0 END), 0)
		FROM likes WHERE comment_id = ?
	`, commentID).Scan(&likes, &dislikes)
	if err != nil {
		return 0, 0, fmt.Errorf("get comment like counts: %w", err)
	}
	return
}

func getUserCommentLike(db *sql.DB, userID, commentID int64) (int, error) {
	var t int
	err := db.QueryRow(
		"SELECT type FROM likes WHERE user_id = ? AND comment_id = ?",
		userID, commentID,
	).Scan(&t)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get user comment like: %w", err)
	}
	return t, nil
}
