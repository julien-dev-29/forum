package sqlite

import (
	"database/sql"
	"fmt"
	"log"

	"forum/internal/crypto"

	_ "github.com/mattn/go-sqlite3"
)

var encryptor *crypto.Encryptor

func InitEncryption(key string) error {
	var err error
	encryptor, err = crypto.NewEncryptor(key)
	if err != nil {
		return fmt.Errorf("init encryption: %w", err)
	}
	log.Println("database encryption enabled")
	return nil
}

func encryptEmail(email string) (encrypted, hash string, err error) {
	if encryptor == nil {
		return "", "", fmt.Errorf("encryption not initialized")
	}
	enc, err := encryptor.Encrypt(email)
	if err != nil {
		return "", "", fmt.Errorf("encrypt email: %w", err)
	}
	return enc, crypto.HashEmail(email), nil
}

func decryptEmail(encrypted string) (string, error) {
	if encryptor == nil {
		return "", fmt.Errorf("encryption not initialized")
	}
	return encryptor.Decrypt(encrypted)
}

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}

func InitSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL,
		username TEXT NOT NULL,
		password TEXT,
		oauth_provider TEXT,
		oauth_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(email)
	);
	`
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("init schema users: %w", err)
	}

	// Migrate existing database: add columns if they don't exist
	migrations := []string{
		"ALTER TABLE users ADD COLUMN oauth_provider TEXT",
		"ALTER TABLE users ADD COLUMN oauth_id TEXT",
		"ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user'",
		"ALTER TABLE posts ADD COLUMN image_path TEXT",
		"ALTER TABLE users ADD COLUMN email_encrypted TEXT",
		"ALTER TABLE users ADD COLUMN email_hash TEXT",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oauth ON users(oauth_provider, oauth_id)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_hash ON users(email_hash)",
	}
	for _, m := range migrations {
		db.Exec(m) // ignore errors - column/index may already exist
	}

	if encryptor != nil {
		backfillEmails(db)
	}

	otherTables := `
	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token TEXT UNIQUE NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		image_path TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS post_categories (
		post_id INTEGER NOT NULL,
		category_id INTEGER NOT NULL,
		PRIMARY KEY (post_id, category_id),
		FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
		FOREIGN KEY (category_id) REFERENCES categories(id)
	);

	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS likes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		post_id INTEGER,
		comment_id INTEGER,
		type INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
		FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE,
		CHECK (
			(post_id IS NOT NULL AND comment_id IS NULL) OR
			(post_id IS NULL AND comment_id IS NOT NULL)
		)
	);

	CREATE UNIQUE INDEX IF NOT EXISTS idx_likes_post ON likes(user_id, post_id) WHERE post_id IS NOT NULL;
	CREATE UNIQUE INDEX IF NOT EXISTS idx_likes_comment ON likes(user_id, comment_id) WHERE comment_id IS NOT NULL;

	CREATE TABLE IF NOT EXISTS notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		actor_id INTEGER NOT NULL,
		type TEXT NOT NULL,
		post_id INTEGER NOT NULL,
		comment_id INTEGER,
		is_read INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
		FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS reports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		reporter_id INTEGER NOT NULL,
		post_id INTEGER,
		comment_id INTEGER,
		reason TEXT NOT NULL,
		custom_text TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		admin_response TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (reporter_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
		FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE,
		CHECK (
			(post_id IS NOT NULL AND comment_id IS NULL) OR
			(post_id IS NULL AND comment_id IS NOT NULL)
		)
	);

	CREATE TABLE IF NOT EXISTS mod_requests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL UNIQUE,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	`
	_, err = db.Exec(otherTables)
	if err != nil {
		return fmt.Errorf("init schema other tables: %w", err)
	}
	return nil
}

func backfillEmails(db *sql.DB) {
	rows, err := db.Query("SELECT id, email FROM users WHERE email_hash IS NULL AND email IS NOT NULL")
	if err != nil {
		log.Printf("backfill emails query: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var email string
		if err := rows.Scan(&id, &email); err != nil {
			log.Printf("backfill scan: %v", err)
			continue
		}
		enc, hash, err := encryptEmail(email)
		if err != nil {
			log.Printf("backfill encrypt id=%d: %v", id, err)
			continue
		}
		if _, err := db.Exec("UPDATE users SET email_encrypted = ?, email_hash = ? WHERE id = ?", enc, hash, id); err != nil {
			log.Printf("backfill update id=%d: %v", id, err)
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("backfill rows: %v", err)
	}
}

func SeedCategories(db *sql.DB) error {
	categories := []string{"General", "Technology", "Sports", "Entertainment"}
	for _, name := range categories {
		_, err := db.Exec("INSERT OR IGNORE INTO categories (name) VALUES (?)", name)
		if err != nil {
			return fmt.Errorf("seed category %q: %w", name, err)
		}
	}
	return nil
}
