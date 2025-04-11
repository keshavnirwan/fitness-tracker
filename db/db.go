package db

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

// Initialize database connection
func InitDB() error {
	var err error
	dsn := "root:keshav@tcp(127.0.0.1:3306)/webtech4"
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	return db.Ping()
}

// CreateUser inserts a new user into the database
func CreateUser(username, email, passwordHash, role string) (int64, error) {
	query := `INSERT INTO person (username, email, password_hash, role) VALUES (?, ?, ?, ?)`
	result, err := db.Exec(query, username, email, passwordHash, role)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// ValidateUser checks if the user exists and password matches
func ValidateUser(username, password string) (bool, string, error) {
	var storedPassword, role string
	err := db.QueryRow("SELECT password_hash, role FROM person WHERE username = ?", username).Scan(&storedPassword, &role)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil // user not found
		}
		return false, "", err
	}

	if storedPassword != password {
		return false, "", nil
	}

	return true, role, nil
}

// GetUserRole retrieves the role for a given username
func GetUserRole(username string) (string, error) {
	var role string
	err := db.QueryRow("SELECT role FROM person WHERE username = ?", username).Scan(&role)
	if err != nil {
		return "", err
	}
	return role, nil
}
