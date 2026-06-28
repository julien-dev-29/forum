package sqlite

import (
	"database/sql"
	"fmt"
	"forum/internal/models"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func adminEmail() string {
	return os.Getenv("ADMIN_EMAIL")
}

func CreateUser(db *sql.DB, email, username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	role := "user"
	if email == adminEmail() {
		role = "admin"
	}
	_, err = db.Exec(
		"INSERT INTO users (email, username, password, role) VALUES (?, ?, ?, ?)",
		email, username, string(hash), role,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func CreateOAuthUser(db *sql.DB, email, username, provider, oauthID string) (*models.User, error) {
	role := "user"
	if email == adminEmail() {
		role = "admin"
	}
	res, err := db.Exec(
		"INSERT INTO users (email, username, password, oauth_provider, oauth_id, role) VALUES (?, ?, NULL, ?, ?, ?)",
		email, username, provider, oauthID, role,
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
		Role:          role,
	}, nil
}

func scanUser(row interface{ Scan(...any) error }) (*models.User, error) {
	u := &models.User{}
	var password, oauthProvider, oauthID *string
	err := row.Scan(&u.ID, &u.Email, &u.Username, &password, &oauthProvider, &oauthID, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
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

const selectUsers = "SELECT id, email, username, password, oauth_provider, oauth_id, role, created_at FROM users"

func GetUserByID(db *sql.DB, id int64) (*models.User, error) {
	return scanUser(db.QueryRow(selectUsers+" WHERE id = ?", id))
}

func GetUserByEmail(db *sql.DB, email string) (*models.User, error) {
	return scanUser(db.QueryRow(selectUsers+" WHERE email = ?", email))
}

func GetUserByOAuth(db *sql.DB, provider, oauthID string) (*models.User, error) {
	return scanUser(db.QueryRow(selectUsers+" WHERE oauth_provider = ? AND oauth_id = ?", provider, oauthID))
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

func GetAllUsers(db *sql.DB) ([]models.User, error) {
	rows, err := db.Query(selectUsers + " ORDER BY username")
	if err != nil {
		return nil, fmt.Errorf("get all users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return users, nil
}

func UpdateUserRole(db *sql.DB, userID int64, role string) error {
	res, err := db.Exec("UPDATE users SET role = ? WHERE id = ?", role, userID)
	if err != nil {
		return fmt.Errorf("update user role: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func CreateModRequest(db *sql.DB, userID int64) error {
	_, err := db.Exec(
		"INSERT INTO mod_requests (user_id, status) VALUES (?, 'pending')",
		userID,
	)
	if err != nil {
		return fmt.Errorf("create mod request: %w", err)
	}
	return nil
}

func GetModRequestByUser(db *sql.DB, userID int64) (*models.ModRequest, error) {
	m := &models.ModRequest{}
	err := db.QueryRow(`
		SELECT mr.id, mr.user_id, u.username, mr.status, mr.created_at
		FROM mod_requests mr
		JOIN users u ON u.id = mr.user_id
		WHERE mr.user_id = ?
	`, userID).Scan(&m.ID, &m.UserID, &m.Username, &m.Status, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get mod request by user: %w", err)
	}
	return m, nil
}

func GetModRequests(db *sql.DB, status string) ([]models.ModRequest, error) {
	rows, err := db.Query(`
		SELECT mr.id, mr.user_id, u.username, mr.status, mr.created_at
		FROM mod_requests mr
		JOIN users u ON u.id = mr.user_id
		WHERE mr.status = ?
		ORDER BY mr.created_at DESC
	`, status)
	if err != nil {
		return nil, fmt.Errorf("get mod requests: %w", err)
	}
	defer rows.Close()

	var requests []models.ModRequest
	for rows.Next() {
		var m models.ModRequest
		if err := rows.Scan(&m.ID, &m.UserID, &m.Username, &m.Status, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan mod request: %w", err)
		}
		requests = append(requests, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return requests, nil
}

func GetAllModRequests(db *sql.DB) ([]models.ModRequest, error) {
	rows, err := db.Query(`
		SELECT mr.id, mr.user_id, u.username, mr.status, mr.created_at
		FROM mod_requests mr
		JOIN users u ON u.id = mr.user_id
		ORDER BY mr.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("get all mod requests: %w", err)
	}
	defer rows.Close()

	var requests []models.ModRequest
	for rows.Next() {
		var m models.ModRequest
		if err := rows.Scan(&m.ID, &m.UserID, &m.Username, &m.Status, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan mod request: %w", err)
		}
		requests = append(requests, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return requests, nil
}

func UpdateModRequestStatus(db *sql.DB, requestID int64, status string) error {
	_, err := db.Exec("UPDATE mod_requests SET status = ? WHERE id = ?", status, requestID)
	if err != nil {
		return fmt.Errorf("update mod request status: %w", err)
	}
	return nil
}

func GetModRequestByID(db *sql.DB, requestID int64) (*models.ModRequest, error) {
	m := &models.ModRequest{}
	err := db.QueryRow(`
		SELECT mr.id, mr.user_id, u.username, mr.status, mr.created_at
		FROM mod_requests mr
		JOIN users u ON u.id = mr.user_id
		WHERE mr.id = ?
	`, requestID).Scan(&m.ID, &m.UserID, &m.Username, &m.Status, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get mod request by id: %w", err)
	}
	return m, nil
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
