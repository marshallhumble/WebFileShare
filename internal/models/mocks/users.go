package mocks

import (
	"fileshare/internal/models"
)

type UserModel struct {
}

func (m *UserModel) Insert(name, email, password string, admin, guest bool) error {
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

func (m *UserModel) Exists(id int) (exist bool, admin bool, user bool, guest bool, disabled bool, error error) {
	switch id {
	case 1:
		return true, false, false, false, false, nil
	default:
		return false, false, false, false, false, nil
	}
}

func (m *UserModel) AdminPageInsert(name, email, password string, admin bool) error {
	return nil
}

func (m *UserModel) GetAllUsers() ([]models.User, error) {
	return nil, nil
}

func (m *UserModel) Get(id int) (models.User, error) {
	var u models.User
	return u, nil
}

func (m *UserModel) UpdateUser(id int, name, email, password string, admin bool) (models.User, error) {
	return models.User{}, nil
}
