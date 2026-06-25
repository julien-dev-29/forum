package sqlite

import (
	"database/sql"
	"fmt"
	"forum/internal/models"
)

func GetUserCommentsWithPosts(db *sql.DB, userID int64) ([]models.UserComment, error) {
	rows, err := db.Query(`
		SELECT c.id, c.content, c.created_at, p.id, p.title
		FROM comments c
		JOIN posts p ON p.id = c.post_id
		WHERE c.user_id = ?
		ORDER BY c.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("get user comments with posts: %w", err)
	}
	defer rows.Close()

	var comments []models.UserComment
	for rows.Next() {
		var uc models.UserComment
		var createdAt string
		if err := rows.Scan(&uc.CommentID, &uc.Content, &createdAt, &uc.PostID, &uc.PostTitle); err != nil {
			return nil, fmt.Errorf("scan user comment: %w", err)
		}
		uc.CreatedAt = parseTime(createdAt)
		comments = append(comments, uc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	if comments == nil {
		comments = []models.UserComment{}
	}
	return comments, nil
}

func GetUserLikesWithPosts(db *sql.DB, userID int64) ([]models.UserLike, error) {
	rows, err := db.Query(`
		SELECT l.type, COALESCE(p.id, 0), COALESCE(p.title, ''), l.created_at, l.comment_id
		FROM likes l
		LEFT JOIN posts p ON p.id = l.post_id
		WHERE l.user_id = ?
		ORDER BY l.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("get user likes: %w", err)
	}
	defer rows.Close()

	var likes []models.UserLike
	for rows.Next() {
		var ul models.UserLike
		var createdAt string
		if err := rows.Scan(&ul.LikeType, &ul.PostID, &ul.PostTitle, &createdAt, &ul.CommentID); err != nil {
			return nil, fmt.Errorf("scan user like: %w", err)
		}
		ul.CreatedAt = parseTime(createdAt)
		likes = append(likes, ul)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	if likes == nil {
		likes = []models.UserLike{}
	}
	return likes, nil
}
