package models

import (
	"context"
	"database/sql"
	"time"
)

// DBModel type for database connection values
type DBModel struct {
	DB *sql.DB
}

// Models is the wrapper to all models
type Models struct {
	DB DBModel
}

// NewModels returns a model type with db connection pool
func NewModels(db *sql.DB) Models {
	return Models{
		DB: DBModel{DB: db},
	}
}

type Item struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	InventoryLevel int       `json:"inventory_level"`
	Price          int       `json:"price"`
	CreatedAt      time.Time `json:"-"`
	UpdatedAt      time.Time `json:"-"`
}

func (m *DBModel) GetItem(id int) (Item, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var item Item

	row := m.DB.QueryRowContext(ctx, `SELECT id, name FROM items WHERE id = $1`, id)
	err := row.Scan(&item.ID, &item.Name)
	if err != nil {
		return item, err
	}

	return item, nil
}

func (m *DBModel) CreateTables() {
	dropItemTable := `DROP TABLE IF EXISTS items`
	_, err := m.DB.Exec(dropItemTable)
	if err != nil {
		panic("could not drop items table")
	}
	createItemsTable := `CREATE TABLE IF NOT EXISTS items (
		id SERIAL PRIMARY KEY
		name TEXT NOT NULL
	)`

	_, err = m.DB.Exec(createItemsTable)
	if err != nil {
		panic("could not create items table")
	}

}
