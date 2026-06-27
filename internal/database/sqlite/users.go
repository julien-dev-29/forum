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

func CreateOAuthUser(db *sql.DB, email, username, provider, oauthID string) (*models.User, error) {
	res, err := db.Exec(
		"INSERT INTO users (email, username, password, oauth_provider, oauth_id) VALUES (?, ?, NULL, ?, ?)",
		email, username, provider, oauthID,
	)
	if err != nil {
		return nil, fmt.Errorf("create oauth user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}
	return &models.User{
		ID:            id,
		Email:         email,
		Username:      username,
		OAuthProvider: provider,
		OAuthID:       oauthID,
	}, nil
}

func GetUserByID(db *sql.DB, id int64) (*models.User, error) {
	u := &models.User{}
	var password, oauthProvider, oauthID *string
	err := db.QueryRow(
		"SELECT id, email, username, password, oauth_provider, oauth_id, created_at FROM users WHERE id = ?",
		id,
	).Scan(&u.ID, &u.Email, &u.Username, &password, &oauthProvider, &oauthID, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	if password != nil {
		u.Password = *password
	}
	if oauthProvider != nil {
		u.OAuthProvider = *oauthProvider
	}
	if oauthID != nil {
		u.OAuthID = *oauthID
	}
	return u, nil
}

func GetUserByEmail(db *sql.DB, email string) (*models.User, error) {
	u := &models.User{}
	var password, oauthProvider, oauthID *string
	err := db.QueryRow(
		"SELECT id, email, username, password, oauth_provider, oauth_id, created_at FROM users WHERE email = ?",
		email,
	).Scan(&u.ID, &u.Email, &u.Username, &password, &oauthProvider, &oauthID, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	if password != nil {
		u.Password = *password
	}
	if oauthProvider != nil {
		u.OAuthProvider = *oauthProvider
	}
	if oauthID != nil {
		u.OAuthID = *oauthID
	}
	return u, nil
}

func GetUserByOAuth(db *sql.DB, provider, oauthID string) (*models.User, error) {
	u := &models.User{}
	var password, oauthProvider, oauthIDStr *string
	err := db.QueryRow(
		"SELECT id, email, username, password, oauth_provider, oauth_id, created_at FROM users WHERE oauth_provider = ? AND oauth_id = ?",
		provider, oauthID,
	).Scan(&u.ID, &u.Email, &u.Username, &password, &oauthProvider, &oauthIDStr, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by oauth: %w", err)
	}
	if password != nil {
		u.Password = *password
	}
	if oauthProvider != nil {
		u.OAuthProvider = *oauthProvider
	}
	if oauthIDStr != nil {
		u.OAuthID = *oauthIDStr
	}
	return u, nil
}

func SetUserOAuth(db *sql.DB, userID int64, provider, oauthID string) error {
	_, err := db.Exec(
		"UPDATE users SET oauth_provider = ?, oauth_id = ? WHERE id = ?",
		provider, oauthID, userID,
	)
	if err != nil {
		return fmt.Errorf("set user oauth: %w", err)
	}
	return nil
}

func AuthenticateUser(db *sql.DB, email, password string) (*models.User, error) {
	u, err := GetUserByEmail(db, email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	if u.Password == "" {
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
