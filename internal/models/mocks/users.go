package mocks

import (
	"fileshare/internal/models"
)

type UserModel struct {
}

func (m *UserModel) Insert(name, email, password string) error {
	switch email {
	case "dupe@example.com":
		return models.ErrDuplicateEmail
	default:
		return nil
	}
}

func (m *UserModel) Authenticate(email, password string) (int, error) {
	if email == "alice@example.com" && password == "pa$$word" {
		return 1, nil
	}

	return 0, models.ErrInvalidCredentials
}

func (m *UserModel) Exists(id int) (bool, bool, error) {
	switch id {
	case 1:
		return true, false, nil
	default:
		return false, false, nil
	}
}

func (m *UserModel) AdminPageInsert(name, email, password string, admin bool) error {
	return nil
}

func (m *UserModel) GetAllUsers() ([]models.User, error) {
	return nil, nil
}
