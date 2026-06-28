package sqlite

import (
	"database/sql"
	"fmt"
	"forum/internal/models"
)

func GetAllCategories(db *sql.DB) ([]models.Category, error) {
	rows, err := db.Query("SELECT id, name FROM categories ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("get all categories: %w", err)
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	if categories == nil {
		categories = []models.Category{}
	}
	return categories, nil
}

func CreateCategory(db *sql.DB, name string) error {
	_, err := db.Exec("INSERT INTO categories (name) VALUES (?)", name)
	if err != nil {
		return fmt.Errorf("create category: %w", err)
	}
	return nil
}

func DeleteCategory(db *sql.DB, id int64) error {
	res, err := db.Exec("DELETE FROM categories WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("category not found")
	}
	return nil
}
