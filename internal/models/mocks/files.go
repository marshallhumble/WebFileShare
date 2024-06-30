package mocks

import (
	"time"

	"fileshare/internal/models"
)

var mockFile = models.SharedFile{
	Id:             1,
	DocName:        "Big Important Document",
	SenderEmail:    "Abar@example.com",
	SenderName:     "Cheryl Smith",
	RecipientEmail: "foo@bar.com",
	RecipientName:  "Susan Smith",
	CreatedAt:      time.Now(),
	Expires:        time.Now().Add(24 * time.Hour),
}

type SharedFileModel struct{}

func (m *SharedFileModel) Insert(docName, recipientUserName, senderUserName, senderEmail, recipientEmail string,
	expiresAt int) (int, error) {
	return 2, nil
}

func (m *SharedFileModel) Get(id int) (models.SharedFile, error) {
	switch id {
	case 1:
		return mockFile, nil
	default:
		return models.SharedFile{}, models.ErrNoRecord
	}
}

func (m *SharedFileModel) Latest() ([]models.SharedFile, error) {
	return []models.SharedFile{mockFile}, nil
}
