package models

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	//External
	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

type UserModelInterface interface {
	Insert(name, email, password string, admin, user, guest, disabled bool) error
	Authenticate(email, password string) (int, error)
	Exists(id int) (exist bool, admin bool, user bool, guest bool, disabled bool, error error)
	GetAllUsers() ([]User, error)
	Get(id int) (User, error)
	UpdateUser(id int, name, email, password string, admin, user, guest bool) (User, error)
	DeleteUser(id int) error
}

type User struct {
	ID             int
	Name           string
	Email          string
	HashedPassword []byte
	Created        time.Time
	Admin          bool
	User           bool
	Guest          bool
	Disabled       bool
}

type UserModel struct {
	DB *sql.DB
}

// Insert The usual user page sign-up no admins can be created this way explicitly declaring it false
func (m *UserModel) Insert(name, email, password string, admin, user, guest, disabled bool) error {
	// Create a bcrypt hash of the plain-text password.
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return err
	}

	stmt := `INSERT INTO users (name, email, hashed_password, created, Admin, user, guest, disabled)
    VALUES(?, ?, ?, UTC_TIMESTAMP(), ?, ?, ?, ?)`

	// Use the Exec() method to insert the user details and hashed password
	// into the users table.
	_, err = m.DB.Exec(stmt, name, email, string(hashedPassword), admin, user, guest, disabled)
	if err != nil {
		// If this returns an error, we use the errors.As() function to check
		// whether the error has the type *mysql.MySQLError. If it does, the
		// error will be assigned to the mySQLError variable. We can then check
		// if the error relates to our users_uc_email key by
		// checking if the error code equals 1062 and the contents of the error
		// message string. If it does, we return an ErrDuplicateEmail error.
		var mySQLError *mysql.MySQLError
		if errors.As(err, &mySQLError) {
			if mySQLError.Number == 1062 && strings.Contains(mySQLError.Message, "users_uc_email") {
				return ErrDuplicateEmail
			}
		}
		return err
	}

	return nil
}

func (m *UserModel) Authenticate(email, password string) (int, error) {
	// Retrieve the id and hashed password associated with the given email. If
	// no matching email exists we return the ErrInvalidCredentials error.
	var id int
	var hashedPassword []byte

	stmt := "SELECT id, hashed_password FROM users WHERE email = ?"

	err := m.DB.QueryRow(stmt, email).Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	// Check whether the hashed password and plain-text password provided match.
	// If they don't, we return the ErrInvalidCredentials error.
	// We want to return the same error so that someone can't mine
	// email address searching for valid combo
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	// Otherwise, the password is correct. Return the user ID.
	return id, nil
}

func (m *UserModel) Exists(id int) (exist bool, admin bool, user bool, guest bool, disabled bool, error error) {
	stmt := `SELECT id, Admin, guest, disabled from users WHERE id = ?`

	var u User

	err := m.DB.QueryRow(stmt, id).Scan(&u.ID, &u.Admin, &u.Guest, &u.Disabled)

	if err != nil {
		// If the query returns no rows, then row.Scan() will return a
		// sql.ErrNoRows error. We use the errors.Is() function check for that
		// error specifically, and return our own ErrNoRecord error
		// instead.
		if errors.Is(err, sql.ErrNoRows) {
			return false, false, false, false, false, ErrNoRecord
		}
	}

	if u.Disabled {
		return false, false, false, false, true, nil
	}

	if u.Admin {
		return true, true, false, false, false, nil
	}

	if u.Guest {
		return true, false, false, true, false, nil
	}

	return true, false, true, false, false, nil
}

func (m *UserModel) GetAllUsers() ([]User, error) {
	stmt := "SELECT id, name, email, hashed_password, created, admin, user, guest, disabled FROM users"

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var users []User

	for rows.Next() {
		var u User
		err = rows.Scan(&u.ID, &u.Name, &u.Email, &u.HashedPassword, &u.Created,
			&u.Admin, &u.User, &u.Guest, &u.Disabled)
		if err != nil {
			return nil, err
		}

		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (m *UserModel) Get(id int) (User, error) {
	stmt := `SELECT id, name, email, created, admin, user, guest, disabled FROM users WHERE id = ?`

	var u User

	err := m.DB.QueryRow(stmt, id).Scan(&u.ID, &u.Name, &u.Email, &u.Created,
		&u.Admin, &u.User, &u.Guest, &u.Disabled)

	if err != nil {
		// If the query returns no rows, then row.Scan() will return a
		// sql.ErrNoRows error. We use the errors.Is() function check for that
		// error specifically, and return our own ErrNoRecord error
		// instead.
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNoRecord
		} else {
			return User{}, err
		}
	}

	return u, nil
}

func (m *UserModel) UpdateUser(id int, name, email, password string, admin, user, guest bool) (User, error) {
	var usr User

	HashedPassword, err := hashPassword(password)
	if err != nil {
		return usr, err
	}

	stmt := `UPDATE users SET name = ?, email = ?, hashed_password = ?, admin = ?, user = ?, guest = ? WHERE id = ?`
	_, err = m.DB.Exec(stmt, name, email, HashedPassword, admin, user, guest, id)
	if err != nil {
		// If this returns an error, we use the errors.As() function to check
		// whether the error has the type *mysql.MySQLError. If it does, the
		// error will be assigned to the mySQLError variable. We can then check
		// if the error relates to our users_uc_email key by
		// checking if the error code equals 1062 and the contents of the error
		// message string. If it does, we return an ErrDuplicateEmail error.
		var mySQLError *mysql.MySQLError
		if errors.As(err, &mySQLError) {
			if mySQLError.Number == 1062 && strings.Contains(mySQLError.Message, "users_uc_email") {
				return usr, ErrDuplicateEmail
			}
		}
		return usr, err
	}

	usr.Name = name
	usr.Email = email

	return usr, nil
}

func (m *UserModel) DeleteUser(id int) error {
	stmt := `DELETE FROM users WHERE id = ?`
	_, err := m.DB.Exec(stmt, id)
	if err != nil {
		return err
	}

	return nil
}

func hashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), 14)
}
