package sqlite

import (
	"database/sql"
	"fmt"
	"forum/internal/models"
)

func GetNotificationActor(db *sql.DB, userID, actorID, postID int64, commentID *int64) (*models.Notification, error) {
	query := `SELECT id, user_id, actor_id, type, post_id, comment_id, is_read FROM notifications
		WHERE user_id = ? AND actor_id = ? AND post_id = ?`
	args := []any{userID, actorID, postID}
	if commentID != nil {
		query += " AND comment_id = ?"
		args = append(args, *commentID)
	} else {
		query += " AND comment_id IS NULL"
	}

	n := &models.Notification{}
	err := db.QueryRow(query, args...).Scan(&n.ID, &n.UserID, &n.ActorID, &n.Type, &n.PostID, &n.CommentID, &n.IsRead)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get notification actor: %w", err)
	}
	return n, nil
}

func UpsertNotification(db *sql.DB, userID, actorID int64, notifType string, postID int64, commentID *int64) error {
	existing, err := GetNotificationActor(db, userID, actorID, postID, commentID)
	if err != nil {
		return err
	}

	if existing != nil {
		if existing.Type != notifType {
			_, err := db.Exec(
				"UPDATE notifications SET type = ?, is_read = 0 WHERE id = ?",
				notifType, existing.ID,
			)
			if err != nil {
				return fmt.Errorf("update notification: %w", err)
			}
		}
		return nil
	}

	var commentIDVal *int64
	if commentID != nil {
		commentIDVal = commentID
	}
	_, err = db.Exec(
		"INSERT INTO notifications (user_id, actor_id, type, post_id, comment_id) VALUES (?, ?, ?, ?, ?)",
		userID, actorID, notifType, postID, commentIDVal,
	)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	return nil
}

func DeleteNotification(db *sql.DB, userID, actorID, postID int64, commentID *int64) error {
	query := "DELETE FROM notifications WHERE user_id = ? AND actor_id = ? AND post_id = ?"
	args := []any{userID, actorID, postID}
	if commentID != nil {
		query += " AND comment_id = ?"
		args = append(args, *commentID)
	} else {
		query += " AND comment_id IS NULL"
	}
	_, err := db.Exec(query, args...)
	return err
}

func GetNotificationsByUser(db *sql.DB, userID int64) ([]models.Notification, error) {
	rows, err := db.Query(`
		SELECT n.id, n.user_id, n.actor_id, u.username, n.type,
			n.post_id, COALESCE(p.title, ''), n.comment_id,
			COALESCE(c.content, ''), n.is_read, n.created_at
		FROM notifications n
		JOIN users u ON u.id = n.actor_id
		LEFT JOIN posts p ON p.id = n.post_id
		LEFT JOIN comments c ON c.id = n.comment_id
		WHERE n.user_id = ?
		ORDER BY n.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("get notifications: %w", err)
	}
	defer rows.Close()

	var notifs []models.Notification
	for rows.Next() {
		var n models.Notification
		var isReadInt int
		var createdAt string
		if err := rows.Scan(&n.ID, &n.UserID, &n.ActorID, &n.ActorName, &n.Type,
			&n.PostID, &n.PostTitle, &n.CommentID, &n.CommentContent, &isReadInt, &createdAt); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		n.IsRead = isReadInt == 1
		n.CreatedAt = parseTime(createdAt)
		notifs = append(notifs, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	if notifs == nil {
		notifs = []models.Notification{}
	}
	return notifs, nil
}

func GetUnreadNotificationCount(db *sql.DB, userID int64) (int, error) {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM notifications WHERE user_id = ? AND is_read = 0",
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get unread count: %w", err)
	}
	return count, nil
}

func MarkNotificationsRead(db *sql.DB, userID int64) error {
	_, err := db.Exec(
		"UPDATE notifications SET is_read = 1 WHERE user_id = ? AND is_read = 0",
		userID,
	)
	return err
}

func GetPostAuthorID(db *sql.DB, postID int64) (int64, error) {
	var userID int64
	err := db.QueryRow("SELECT user_id FROM posts WHERE id = ?", postID).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("get post author: %w", err)
	}
	return userID, nil
}

func GetCommentAuthorID(db *sql.DB, commentID int64) (int64, error) {
	var userID int64
	err := db.QueryRow("SELECT user_id FROM comments WHERE id = ?", commentID).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("get comment author: %w", err)
	}
	return userID, nil
}
