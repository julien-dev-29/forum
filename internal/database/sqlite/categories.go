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
