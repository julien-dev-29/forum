package models

import (
	"database/sql"
	"errors"
	"time"
)

type Post struct {
	ID      int
	Title   string
	Content string
	Created string
	Updated string
}

type PostModel struct {
	DB *sql.DB
}

func (m *PostModel) Insert(title string, content string) (int, error) {
	stmt := `INSERT INTO posts (title, content, created_at) VALUES (?, ?, ?)`
	result, err := m.DB.Exec(stmt, title, content, time.Now())
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (m *PostModel) Get(id int) (*Post, error) {
	stmt := `SELECT id, title, content, created_at  FROM posts
 WHERE id = ?`
	p := &Post{}
	err := m.DB.QueryRow(stmt, id).Scan(&p.ID, &p.Title, &p.Content, &p.Created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		} else {
			return nil, err
		}
	}
	return p, nil
}
func (m *PostModel) GetAll() ([]*Post, error) {
	stmt := `SELECT id, title, content, created_at  FROM posts`
	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts := []*Post{}
	for rows.Next() {
		p := &Post{}
		err = rows.Scan(&p.ID, &p.Title, &p.Content, &p.Created)
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}

func (m *PostModel) Latest() ([]*Post, error) {
	stmt := `SELECT id, title, content, created_at FROM posts
 ORDER BY created_at DESC LIMIT 10`
	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []*Post{}

	for rows.Next() {
		p := &Post{}
		err = rows.Scan(&p.ID, &p.Title, &p.Content, &p.Created)
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}
