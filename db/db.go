package db

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

// InitDB initializes the global DB connection
func InitDB() error {
	var err error
	dsn := "root:keshav@tcp(127.0.0.1:3306)/webtech4"
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	return db.Ping()
}

// CreateUser inserts a new user into the person table
func CreateUser(username, email, passwordHash, role string) (int64, error) {
	query := `INSERT INTO person (username, email, password_hash, role) VALUES (?, ?, ?, ?)`
	result, err := db.Exec(query, username, email, passwordHash, role)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// InsertUserInfo adds personal details into the user_info table
func InsertUserInfo(userID int64, fullName string, age int, gender string, height, weight float64) error {
	query := `INSERT INTO user_info (user_id, full_name, age, gender, height_cm, weight_kg) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(query, userID, fullName, age, gender, height, weight)
	return err
}

// SaveUserInfoByID inserts or updates personal info using user ID
func SaveUserInfoByID(userID int64, fullName string, age int, gender string, height, weight float64) error {
	query := `
		INSERT INTO user_info (user_id, full_name, age, gender, height_cm, weight_kg)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			full_name = VALUES(full_name),
			age = VALUES(age),
			gender = VALUES(gender),
			height_cm = VALUES(height_cm),
			weight_kg = VALUES(weight_kg)
	`
	_, err := db.Exec(query, userID, fullName, age, gender, height, weight)
	return err
}

// SaveUserInfo inserts or updates personal info using username
func SaveUserInfo(username string, fullName string, age int, gender string, height, weight float64) error {
	// Get user ID from username
	var userID int
	err := db.QueryRow("SELECT id FROM person WHERE username = ?", username).Scan(&userID)
	if err != nil {
		return err
	}

	// Check if user_info entry exists for this user
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM user_info WHERE user_id = ?)", userID).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		// Update
		_, err = db.Exec(`UPDATE user_info SET full_name=?, age=?, gender=?, height_cm=?, weight_kg=? WHERE user_id=?`,
			fullName, age, gender, height, weight, userID)
	} else {
		// Insert
		_, err = db.Exec(`INSERT INTO user_info (user_id, full_name, age, gender, height_cm, weight_kg) VALUES (?, ?, ?, ?, ?, ?)`,
			userID, fullName, age, gender, height, weight)
	}

	return err
}

// ValidateUser checks if the username/password is valid
func ValidateUser(username, password string) (bool, string, error) {
	var storedPassword, role string
	err := db.QueryRow("SELECT password_hash, role FROM person WHERE username = ?", username).Scan(&storedPassword, &role)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", err
	}

	if storedPassword != password {
		return false, "", nil
	}

	return true, role, nil
}

// UserInfo holds personal data
type UserInfo struct {
	FullName string
	Age      int
	Gender   string
	Height   float64
	Weight   float64
}

// GetUserInfoByUsername fetches user_info using username
func GetUserInfoByUsername(username string) (*UserInfo, error) {
	query := `
		SELECT ui.full_name, ui.age, ui.gender, ui.height_cm, ui.weight_kg
		FROM user_info ui
		JOIN person p ON ui.user_id = p.id
		WHERE p.username = ?
	`
	var info UserInfo
	err := db.QueryRow(query, username).Scan(&info.FullName, &info.Age, &info.Gender, &info.Height, &info.Weight)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// SaveOrUpdateProgress inserts or updates daily progress
func SaveOrUpdateProgress(userID int64, workout, meals, water bool) error {
	query := `
	INSERT INTO user_progress (user_id, date, workout_done, meals_logged, water_done)
	VALUES (?, CURDATE(), ?, ?, ?)
	ON DUPLICATE KEY UPDATE 
		workout_done = VALUES(workout_done),
		meals_logged = VALUES(meals_logged),
		water_done = VALUES(water_done)
	`
	_, err := db.Exec(query, userID, workout, meals, water)
	return err
}

// GetTodayProgress retrieves today's progress
func GetTodayProgress(userID int64) (bool, bool, bool, error) {
	var workout, meals, water bool
	query := `SELECT workout_done, meals_logged, water_done FROM user_progress WHERE user_id = ? AND date = CURDATE()`
	err := db.QueryRow(query, userID).Scan(&workout, &meals, &water)
	if err == sql.ErrNoRows {
		return false, false, false, nil // No progress yet
	}
	if err != nil {
		return false, false, false, err
	}
	return workout, meals, water, nil
}
func GetUserIDByUsername(username string) (int64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM person WHERE username = ?", username).Scan(&id)
	return id, err
}

func SendMessage(senderID, receiverID int64, content string) error {
	query := `INSERT INTO messages (sender_id, receiver_id, message) VALUES (?, ?, ?)`
	_, err := db.Exec(query, senderID, receiverID, content)
	return err
}

type Message struct {
	Sender  string
	Content string
	Time    string
}
