package db

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// User represents an application user.
type User struct {
	ID           int64     `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// CreateUser inserts a new user into the database.
func CreateUser(exec sqlx.ExtContext, user *User) (int64, error) {
	result, err := exec.ExecContext(context.Background(), `
        INSERT INTO users (email, password_hash, created_at)
        VALUES (?, ?, ?)
    `, user.Email, user.PasswordHash, user.CreatedAt)
	if err != nil {
		return 0, handleError(err, "failed to create user")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, handleError(err, "failed to get inserted user ID")
	}
	user.ID = id
	return id, nil
}

// GetUserByEmail retrieves a user by their email address.
func GetUserByEmail(db *sqlx.DB, email string) (*User, error) {
	var u User
	err := db.GetContext(context.Background(), &u, `SELECT * FROM users WHERE email = ?`, email)
	if err != nil {
		return nil, handleError(err, "failed to fetch user by email")
	}
	return &u, nil
}

// GetUserByID retrieves a user by ID.
func GetUserByID(db *sqlx.DB, id int64) (*User, error) {
	var u User
	err := db.GetContext(context.Background(), &u, `SELECT * FROM users WHERE id = ?`, id)
	if err != nil {
		return nil, handleError(err, "failed to fetch user by id")
	}
	return &u, nil
}
