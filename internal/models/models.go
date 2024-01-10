package models

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
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

// Item type for all items
type Item struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	InventoryLevel int       `json:"inventory_level"`
	Price          int       `json:"price"`
	Image          string    `json:"image"`
	IsRecurring    bool      `json:"is_recurring"`
	PlanID         string    `json:"plan_id"`
	CreatedAt      time.Time `json:"-"`
	UpdatedAt      time.Time `json:"-"`
}

// Order type for all orders
type Order struct {
	ID            int         `json:"id"`
	ItemID        int         `json:"item_id"`
	TransactionID int         `json:"transaction_id"`
	CustomerID    int         `json:"customer_id"`
	StatusID      int         `json:"status_id"`
	Quantity      int         `json:"quantity"`
	Amount        int         `json:"amount"`
	CreatedAt     time.Time   `json:"-"`
	UpdatedAt     time.Time   `json:"-"`
	Item          Item        `json:"item"`
	Transaction   Transaction `json:"transaction"`
	Customer      Customer    `json:"customer"`
}

// Status type for order statuses
type Status struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// TransactionStatus type for transaction statuses
type TransactionStatus struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// Transaction type for transactions
type Transaction struct {
	ID                  int       `json:"id"`
	Amount              int       `json:"amount"`
	Currency            string    `json:"currency"`
	LastFour            string    `json:"last_four"`
	ExpiryMonth         int       `json:"expiry_month"`
	ExpiryYear          int       `json:"expiry_year"`
	PaymentIntent       string    `json:"payment_intent"`
	PaymentMethod       string    `json:"payment_method"`
	BankReturnCode      string    `json:"bank_return_code"`
	TransactionStatusID int       `json:"transaction_status_id"`
	CreatedAt           time.Time `json:"-"`
	UpdatedAt           time.Time `json:"-"`
}

// User type for transtactions
type User struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// Customer type for transtactions
type Customer struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

func (m *DBModel) GetItem(id int) (Item, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var item Item

	row := m.DB.QueryRowContext(ctx, `
		SELECT id, name, description, inventory_level, price, COALESCE(image, ''), is_recurring, plan_id, created_at, updated_at 
		FROM items 
		WHERE id = $1`, id)
	err := row.Scan(
		&item.ID,
		&item.Name,
		&item.Description,
		&item.InventoryLevel,
		&item.Price,
		&item.Image,
		&item.IsRecurring,
		&item.PlanID,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}

	return item, nil
}

// InsertTransaction inserts a transaction and returns txn ID
func (m *DBModel) InsertTransaction(txn Transaction) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO transactions
		(amount, currency, last_four, bank_return_code, payment_intent, payment_method, transaction_status_id, expiry_month, expiry_year, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	var txnID int
	err := m.DB.QueryRowContext(ctx, query,
		txn.Amount,
		txn.Currency,
		txn.LastFour,
		txn.BankReturnCode,
		txn.PaymentIntent,
		txn.PaymentMethod,
		txn.TransactionStatusID,
		txn.ExpiryMonth,
		txn.ExpiryYear,
		time.Now(),
		time.Now(),
	).Scan(&txnID)
	if err != nil {
		return 0, err
	}

	return txnID, nil
}

// InsertOrder inserts an order and returns order ID
func (m *DBModel) InsertOrder(order Order) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO orders
		(item_id, transaction_id, status_id, customer_id, quantity, amount, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var orderID int
	err := m.DB.QueryRowContext(ctx, query,
		order.ItemID,
		order.TransactionID,
		order.StatusID,
		order.CustomerID,
		order.Quantity,
		order.Amount,
		time.Now(),
		time.Now(),
	).Scan(&orderID)
	if err != nil {
		return 0, err
	}

	return orderID, nil
}

// InsertCustomer inserts a customer and returns order ID
func (m *DBModel) InsertCustomer(c Customer) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO customers
		(first_name, last_name, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var customerID int
	err := m.DB.QueryRowContext(ctx, query,
		c.FirstName,
		c.LastName,
		c.Email,
		time.Now(),
		time.Now(),
	).Scan(&customerID)
	if err != nil {
		return 0, err
	}

	return customerID, nil
}

func (m *DBModel) CreateTables() {
	// dropItemTable := `DROP TABLE IF EXISTS items`
	// _, err := m.DB.Exec(dropItemTable)
	// if err != nil {
	// 	panic("could not drop items table")
	// }
	// createItemsTable := `CREATE TABLE IF NOT EXISTS items (
	// 	id SERIAL PRIMARY KEY,
	// 	name TEXT NOT NULL
	// )`

	// _, err := m.DB.Exec(createItemsTable)
	// if err != nil {
	// 	panic("could not create items table")
	// }

	err := m.testItemData()
	if err != nil {
		panic("could not create test item data")
	}

}

func (m *DBModel) testItemData() error {
	insertTestData := `INSERT INTO items (id, name, description, inventory_level, price, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := m.DB.Exec(insertTestData,
		2,
		"Bronze Series Plan",
		"Receive three unique items for the price of two every month.",
		10,
		2000,
		time.Now(),
		time.Now(),
	)
	if err != nil {
		panic("could not create test item data")
	}

	return nil
}

// GetUserByEmail gets user by email address
func (m *DBModel) GetUserByEmail(email string) (User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	email = strings.ToLower(email)
	var u User

	row := m.DB.QueryRowContext(ctx, `
		SELECT id, first_name, last_name, email, password, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email)

	err := row.Scan(
		&u.ID,
		&u.FirstName,
		&u.LastName,
		&u.Email,
		&u.Password,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return u, err
	}
	return u, nil
}

func (m *DBModel) Authenticate(email, password string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id int
	var hashedPassword string

	row := m.DB.QueryRowContext(ctx, `SELECT id, password FROM users WHERE email = $1`, email)
	err := row.Scan(&id, &hashedPassword)
	if err != nil {
		return id, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return 0, errors.New("incorrect password")
	} else if err != nil {
		return 0, err
	}
	return id, nil
}

func (m *DBModel) UpdatePasswordForUser(u User, hash string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE users SET password = $1 WHERE id = $2`
	_, err := m.DB.ExecContext(ctx, stmt, hash, u.ID)
	if err != nil {
		return err
	}

	return nil
}

func (m *DBModel) GetAllOrders(recurring bool) ([]*Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var orders []*Order

	query := `
	select o.id, o.item_id, o.transaction_id, o.customer_id, o.status_id, o.quantity, o.amount, o.created_at, o.updated_at,
		i.id, i.name, t.id, t.amount, t.currency, t.last_four, t.expiry_month, t.expiry_year, t.payment_intent, t.bank_return_code,
		c.id, c.first_name, c.last_name, c.email
	from orders o
	left join items i on (o.item_id = i.id)
	left join transactions t on (o.transaction_id = t.id)
	left join customers c on (o.customer_id = c.id)
	where i.is_recurring = $1 
	order by o.created_at desc 
	`

	rows, err := m.DB.QueryContext(ctx, query, recurring)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var o Order
		err = rows.Scan(
			&o.ID,
			&o.ItemID,
			&o.TransactionID,
			&o.CustomerID,
			&o.StatusID,
			&o.Quantity,
			&o.Amount,
			&o.CreatedAt,
			&o.UpdatedAt,
			&o.Item.ID,
			&o.Item.Name,
			&o.Transaction.ID,
			&o.Transaction.Amount,
			&o.Transaction.Currency,
			&o.Transaction.LastFour,
			&o.Transaction.ExpiryMonth,
			&o.Transaction.ExpiryYear,
			&o.Transaction.PaymentIntent,
			&o.Transaction.BankReturnCode,
			&o.Customer.ID,
			&o.Customer.FirstName,
			&o.Customer.LastName,
			&o.Customer.Email,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}
	return orders, nil
}

// GetAllOrdersPaginated returns a slice of a subset of orders
func (m *DBModel) GetAllOrdersPaginated(recurring bool, pageSize, page int) ([]*Order, int, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	offset := (page - 1) * pageSize

	var orders []*Order

	query := `
	select o.id, o.item_id, o.transaction_id, o.customer_id, o.status_id, o.quantity, o.amount, o.created_at, o.updated_at,
		i.id, i.name, t.id, t.amount, t.currency, t.last_four, t.expiry_month, t.expiry_year, t.payment_intent, t.bank_return_code,
		c.id, c.first_name, c.last_name, c.email
	from orders o
	left join items i on (o.item_id = i.id)
	left join transactions t on (o.transaction_id = t.id)
	left join customers c on (o.customer_id = c.id)
	where i.is_recurring = $1 
	order by o.created_at desc 
	limit $2 offset $3
	`

	rows, err := m.DB.QueryContext(ctx, query, recurring, pageSize, offset)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var o Order
		err = rows.Scan(
			&o.ID,
			&o.ItemID,
			&o.TransactionID,
			&o.CustomerID,
			&o.StatusID,
			&o.Quantity,
			&o.Amount,
			&o.CreatedAt,
			&o.UpdatedAt,
			&o.Item.ID,
			&o.Item.Name,
			&o.Transaction.ID,
			&o.Transaction.Amount,
			&o.Transaction.Currency,
			&o.Transaction.LastFour,
			&o.Transaction.ExpiryMonth,
			&o.Transaction.ExpiryYear,
			&o.Transaction.PaymentIntent,
			&o.Transaction.BankReturnCode,
			&o.Customer.ID,
			&o.Customer.FirstName,
			&o.Customer.LastName,
			&o.Customer.Email,
		)
		if err != nil {
			return nil, 0, 0, err
		}
		orders = append(orders, &o)
	}
	query = `
		SELECT count(o.id)
		FROM orders o
		LEFT JOIN items i on (o.item_id = i.id)
		WHERE i.is_recurring = $1
	`
	var totalRecords int
	countRow := m.DB.QueryRowContext(ctx, query, recurring)
	err = countRow.Scan(&totalRecords)
	if err != nil {
		return nil, 0, 0, err
	}
	lastPage := totalRecords / pageSize

	return orders, lastPage, totalRecords, nil
}

func (m *DBModel) GetOrderByID(id int) (Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var o Order

	query := `
	select o.id, o.item_id, o.transaction_id, o.customer_id, o.status_id, o.quantity, o.amount, o.created_at, o.updated_at,
		i.id, i.name, t.id, t.amount, t.currency, t.last_four, t.expiry_month, t.expiry_year, t.payment_intent, t.bank_return_code,
		c.id, c.first_name, c.last_name, c.email
	from orders o
	left join items i on (o.item_id = i.id)
	left join transactions t on (o.transaction_id = t.id)
	left join customers c on (o.customer_id = c.id)
	where o.id = $1 
	
	`

	row := m.DB.QueryRowContext(ctx, query, id)

	err := row.Scan(
		&o.ID,
		&o.ItemID,
		&o.TransactionID,
		&o.CustomerID,
		&o.StatusID,
		&o.Quantity,
		&o.Amount,
		&o.CreatedAt,
		&o.UpdatedAt,
		&o.Item.ID,
		&o.Item.Name,
		&o.Transaction.ID,
		&o.Transaction.Amount,
		&o.Transaction.Currency,
		&o.Transaction.LastFour,
		&o.Transaction.ExpiryMonth,
		&o.Transaction.ExpiryYear,
		&o.Transaction.PaymentIntent,
		&o.Transaction.BankReturnCode,
		&o.Customer.ID,
		&o.Customer.FirstName,
		&o.Customer.LastName,
		&o.Customer.Email,
	)
	if err != nil {
		return o, err
	}

	return o, nil
}

func (m *DBModel) UpdateOrderStatus(id, statusID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE orders SET status_id = $1 WHERE id = $2`

	_, err := m.DB.ExecContext(ctx, stmt, statusID, id)
	if err != nil {
		return err
	}
	return nil
}
