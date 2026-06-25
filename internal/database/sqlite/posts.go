package sqlite

import (
	"database/sql"
	"fmt"
	"forum/internal/models"
	"time"
)

func CreatePost(db *sql.DB, userID int64, title, content string, categoryIDs []int64) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"INSERT INTO posts (user_id, title, content) VALUES (?, ?, ?)",
		userID, title, content,
	)
	if err != nil {
		return 0, fmt.Errorf("insert post: %w", err)
	}

	postID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}

	for _, cid := range categoryIDs {
		_, err := tx.Exec(
			"INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)",
			postID, cid,
		)
		if err != nil {
			return 0, fmt.Errorf("insert post category: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	return postID, nil
}

func GetAllPosts(db *sql.DB, userID *int64) ([]models.Post, error) {
	query := `
		SELECT p.id, p.user_id, p.title, p.content, p.created_at, u.username
		FROM posts p
		JOIN users u ON u.id = p.user_id
		ORDER BY p.created_at DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("get all posts: %w", err)
	}
	defer rows.Close()

	return scanPosts(rows, db, userID)
}

func GetPostsByCategory(db *sql.DB, categoryID int64, userID *int64) ([]models.Post, error) {
	query := `
		SELECT p.id, p.user_id, p.title, p.content, p.created_at, u.username
		FROM posts p
		JOIN users u ON u.id = p.user_id
		JOIN post_categories pc ON pc.post_id = p.id
		WHERE pc.category_id = ?
		ORDER BY p.created_at DESC
	`
	rows, err := db.Query(query, categoryID)
	if err != nil {
		return nil, fmt.Errorf("get posts by category: %w", err)
	}
	defer rows.Close()

	return scanPosts(rows, db, userID)
}

func GetPostsByUser(db *sql.DB, targetUserID int64, currentUserID *int64) ([]models.Post, error) {
	rows, err := db.Query(`
		SELECT p.id, p.user_id, p.title, p.content, p.created_at, u.username
		FROM posts p
		JOIN users u ON u.id = p.user_id
		WHERE p.user_id = ?
		ORDER BY p.created_at DESC
	`, targetUserID)
	if err != nil {
		return nil, fmt.Errorf("get posts by user: %w", err)
	}
	defer rows.Close()

	return scanPosts(rows, db, currentUserID)
}

func GetLikedPosts(db *sql.DB, userID int64) ([]models.Post, error) {
	rows, err := db.Query(`
		SELECT p.id, p.user_id, p.title, p.content, p.created_at, u.username
		FROM posts p
		JOIN users u ON u.id = p.user_id
		JOIN likes l ON l.post_id = p.id AND l.user_id = ? AND l.type = 1
		ORDER BY p.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("get liked posts: %w", err)
	}
	defer rows.Close()

	uid := userID
	return scanPosts(rows, db, &uid)
}

func GetPostByID(db *sql.DB, postID int64, userID *int64) (*models.Post, error) {
	p := &models.Post{}
	var createdAt string
	err := db.QueryRow(`
		SELECT p.id, p.user_id, p.title, p.content, p.created_at, u.username
		FROM posts p
		JOIN users u ON u.id = p.user_id
		WHERE p.id = ?
	`, postID).Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &createdAt, &p.Username)
	if err != nil {
		return nil, fmt.Errorf("get post by id: %w", err)
	}
	p.CreatedAt = parseTime(createdAt)

	cats, err := getPostCategories(db, postID)
	if err != nil {
		return nil, err
	}
	p.Categories = cats

	likes, dislikes, err := getPostLikeCounts(db, postID)
	if err != nil {
		return nil, err
	}
	p.LikeCount = likes
	p.DislikeCount = dislikes

	if userID != nil {
		like, err := getUserPostLike(db, *userID, postID)
		if err != nil {
			return nil, err
		}
		p.UserLike = like
	}

	return p, nil
}

func scanPosts(rows *sql.Rows, db *sql.DB, userID *int64) ([]models.Post, error) {
	var posts []models.Post
	for rows.Next() {
		var createdAt string
		p := models.Post{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &createdAt, &p.Username); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		p.CreatedAt = parseTime(createdAt)

		cats, err := getPostCategories(db, p.ID)
		if err != nil {
			return nil, err
		}
		p.Categories = cats

		likes, dislikes, err := getPostLikeCounts(db, p.ID)
		if err != nil {
			return nil, err
		}
		p.LikeCount = likes
		p.DislikeCount = dislikes

		if userID != nil {
			like, err := getUserPostLike(db, *userID, p.ID)
			if err != nil {
				return nil, err
			}
			p.UserLike = like
		}

		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	if posts == nil {
		posts = []models.Post{}
	}
	return posts, nil
}

func UpdatePost(db *sql.DB, postID, userID int64, title, content string, categoryIDs []int64) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"UPDATE posts SET title = ?, content = ? WHERE id = ? AND user_id = ?",
		title, content, postID, userID,
	)
	if err != nil {
		return fmt.Errorf("update post: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("post not found or not owned")
	}

	_, err = tx.Exec("DELETE FROM post_categories WHERE post_id = ?", postID)
	if err != nil {
		return fmt.Errorf("delete post categories: %w", err)
	}

	for _, cid := range categoryIDs {
		_, err := tx.Exec(
			"INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)",
			postID, cid,
		)
		if err != nil {
			return fmt.Errorf("insert post category: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func DeletePost(db *sql.DB, postID, userID int64) error {
	res, err := db.Exec("DELETE FROM posts WHERE id = ? AND user_id = ?", postID, userID)
	if err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("post not found or not owned")
	}
	return nil
}

func parseTime(s string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", s)
	if err == nil {
		return t
	}
	t, err = time.Parse("2006-01-02 15:04:05", s)
	if err == nil {
		return t
	}
	return time.Time{}
}
