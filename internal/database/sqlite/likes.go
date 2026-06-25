package sqlite

import (
	"database/sql"
	"fmt"
)

func TogglePostLike(db *sql.DB, userID, postID int64, likeType int) error {
	existing, err := getUserPostLike(db, userID, postID)
	if err != nil {
		return err
	}

	notifType := "like"
	if likeType == -1 {
		notifType = "dislike"
	}

	postAuthorID, _ := GetPostAuthorID(db, postID)

	if existing == likeType {
		_, err := db.Exec(
			"DELETE FROM likes WHERE user_id = ? AND post_id = ?",
			userID, postID,
		)
		if err == nil && postAuthorID != userID {
			_ = DeleteNotification(db, postAuthorID, userID, postID, nil)
		}
		return err
	}

	if existing != 0 {
		_, err := db.Exec(
			"UPDATE likes SET type = ? WHERE user_id = ? AND post_id = ?",
			likeType, userID, postID,
		)
		if err == nil && postAuthorID != userID {
			_ = UpsertNotification(db, postAuthorID, userID, notifType, postID, nil)
		}
		return err
	}

	_, err = db.Exec(
		"INSERT INTO likes (user_id, post_id, type) VALUES (?, ?, ?)",
		userID, postID, likeType,
	)
	if err == nil && postAuthorID != userID {
		_ = UpsertNotification(db, postAuthorID, userID, notifType, postID, nil)
	}
	return err
}

func ToggleCommentLike(db *sql.DB, userID, commentID int64, likeType int) error {
	existing, err := getUserCommentLike(db, userID, commentID)
	if err != nil {
		return err
	}

	notifType := "like"
	if likeType == -1 {
		notifType = "dislike"
	}

	commentAuthorID, _ := GetCommentAuthorID(db, commentID)

	// For comment likes, we need a postID for the notification
	comment, err := GetCommentByID(db, commentID)
	if err != nil {
		return fmt.Errorf("get comment for notification: %w", err)
	}
	postID := comment.PostID

	if existing == likeType {
		_, err := db.Exec(
			"DELETE FROM likes WHERE user_id = ? AND comment_id = ?",
			userID, commentID,
		)
		cid := commentID
		if err == nil && commentAuthorID != userID {
			_ = DeleteNotification(db, commentAuthorID, userID, postID, &cid)
		}
		return err
	}

	if existing != 0 {
		_, err := db.Exec(
			"UPDATE likes SET type = ? WHERE user_id = ? AND comment_id = ?",
			likeType, userID, commentID,
		)
		cid := commentID
		if err == nil && commentAuthorID != userID {
			_ = UpsertNotification(db, commentAuthorID, userID, notifType, postID, &cid)
		}
		return err
	}

	_, err = db.Exec(
		"INSERT INTO likes (user_id, comment_id, type) VALUES (?, ?, ?)",
		userID, commentID, likeType,
	)
	cid := commentID
	if err == nil && commentAuthorID != userID {
		_ = UpsertNotification(db, commentAuthorID, userID, notifType, postID, &cid)
	}
	return err
}

func HasUserLikedPost(db *sql.DB, userID, postID int64) (bool, error) {
	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM likes WHERE user_id = ? AND post_id = ? AND type = 1)",
		userID, postID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user liked post: %w", err)
	}
	return exists, nil
}

func HasUserDislikedPost(db *sql.DB, userID, postID int64) (bool, error) {
	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM likes WHERE user_id = ? AND post_id = ? AND type = -1)",
		userID, postID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user disliked post: %w", err)
	}
	return exists, nil
}

func HasUserLikedComment(db *sql.DB, userID, commentID int64) (bool, error) {
	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM likes WHERE user_id = ? AND comment_id = ? AND type = 1)",
		userID, commentID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user liked comment: %w", err)
	}
	return exists, nil
}

func HasUserDislikedComment(db *sql.DB, userID, commentID int64) (bool, error) {
	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM likes WHERE user_id = ? AND comment_id = ? AND type = -1)",
		userID, commentID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user disliked comment: %w", err)
	}
	return exists, nil
}
