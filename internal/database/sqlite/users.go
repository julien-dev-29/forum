package sqlite

import (
	"database/sql"
	"fmt"
	"forum/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func CreateUser(db *sql.DB, email, username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	_, err = db.Exec(
		"INSERT INTO users (email, username, password) VALUES (?, ?, ?)",
		email, username, string(hash),
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func GetUserByEmail(db *sql.DB, email string) (*models.User, error) {
	u := &models.User{}
	err := db.QueryRow(
		"SELECT id, email, username, password, created_at FROM users WHERE email = ?",
		email,
	).Scan(&u.ID, &u.Email, &u.Username, &u.Password, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func GetUserByID(db *sql.DB, id int64) (*models.User, error) {
	u := &models.User{}
	err := db.QueryRow(
		"SELECT id, email, username, password, created_at FROM users WHERE id = ?",
		id,
	).Scan(&u.ID, &u.Email, &u.Username, &u.Password, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func AuthenticateUser(db *sql.DB, email, password string) (*models.User, error) {
	u, err := GetUserByEmail(db, email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	return u, nil
}

func EmailExists(db *sql.DB, email string) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check email: %w", err)
	}
	return exists, nil
}

func getPostCategories(db *sql.DB, postID int64) ([]models.Category, error) {
	rows, err := db.Query(`
		SELECT c.id, c.name FROM categories c
		JOIN post_categories pc ON pc.category_id = c.id
		WHERE pc.post_id = ?
	`, postID)
	if err != nil {
		return nil, fmt.Errorf("get post categories: %w", err)
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func getPostLikeCounts(db *sql.DB, postID int64) (likes, dislikes int, err error) {
	err = db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN type = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = -1 THEN 1 ELSE 0 END), 0)
		FROM likes WHERE post_id = ?
	`, postID).Scan(&likes, &dislikes)
	if err != nil {
		return 0, 0, fmt.Errorf("get post like counts: %w", err)
	}
	return
}

func getUserPostLike(db *sql.DB, userID, postID int64) (int, error) {
	var t int
	err := db.QueryRow(
		"SELECT type FROM likes WHERE user_id = ? AND post_id = ?",
		userID, postID,
	).Scan(&t)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get user post like: %w", err)
	}
	return t, nil
}
