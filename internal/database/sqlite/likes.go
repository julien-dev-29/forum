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

	if existing == likeType {
		_, err := db.Exec(
			"DELETE FROM likes WHERE user_id = ? AND post_id = ?",
			userID, postID,
		)
		return err
	}

	if existing != 0 {
		_, err := db.Exec(
			"UPDATE likes SET type = ? WHERE user_id = ? AND post_id = ?",
			likeType, userID, postID,
		)
		return err
	}

	_, err = db.Exec(
		"INSERT INTO likes (user_id, post_id, type) VALUES (?, ?, ?)",
		userID, postID, likeType,
	)
	return err
}

func ToggleCommentLike(db *sql.DB, userID, commentID int64, likeType int) error {
	existing, err := getUserCommentLike(db, userID, commentID)
	if err != nil {
		return err
	}

	if existing == likeType {
		_, err := db.Exec(
			"DELETE FROM likes WHERE user_id = ? AND comment_id = ?",
			userID, commentID,
		)
		return err
	}

	if existing != 0 {
		_, err := db.Exec(
			"UPDATE likes SET type = ? WHERE user_id = ? AND comment_id = ?",
			likeType, userID, commentID,
		)
		return err
	}

	_, err = db.Exec(
		"INSERT INTO likes (user_id, comment_id, type) VALUES (?, ?, ?)",
		userID, commentID, likeType,
	)
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
