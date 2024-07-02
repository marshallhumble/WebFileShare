package models

import (
	"database/sql"
	"errors"
	"time"
)

type SharedFileModelInterface interface {
	Insert(docName, senderUserName, senderEmail, recipientUserName, recipientEmail,
		password string, expiresAt int) (int, error)
	Get(id int) (SharedFile, error)
	Latest() ([]SharedFile, error)
}

type SharedFile struct {
	Id             int
	DocName        string
	SenderName     string
	SenderEmail    string
	RecipientName  string
	RecipientEmail string
	Password       string
	CreatedAt      time.Time
	Expires        time.Time
}

type SharedFileModel struct {
	DB *sql.DB
}

func (m *SharedFileModel) Insert(docName, senderUserName, senderEmail, recipientUserName, recipientEmail,
	password string, expiresAt int) (int, error) {
	stmt := `INSERT INTO files (DocName, SenderName, SenderEmail, RecipientName, RecipientEmail, Password,
                  CreatedAt, Expires) 
VALUES (?, ?, ?, ?, ?, ?, UTC_TIMESTAMP(), DATE_ADD(UTC_TIMESTAMP(), INTERVAL ? DAY))`

	result, err := m.DB.Exec(stmt, docName, senderUserName, senderEmail, recipientUserName, recipientEmail,
		password, expiresAt)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (m *SharedFileModel) Get(id int) (SharedFile, error) {
	stmt := `SELECT Id, DocName, RecipientName, SenderName, CreatedAt, 
       SenderEmail, RecipientEmail FROM files WHERE Expires > UTC_TIMESTAMP() AND id = ?`

	var s SharedFile

	if err := m.DB.QueryRow(stmt, id).Scan(&s.Id, &s.DocName, &s.RecipientName, &s.SenderName, &s.CreatedAt,
		&s.SenderEmail, &s.RecipientEmail); err != nil {
		// If the query returns no rows, then row.Scan() will return a
		// sql.ErrNoRows error. We use the errors.Is() function check for that
		// error specifically, and return our own ErrNoRecord error
		// instead.
		if errors.Is(err, sql.ErrNoRows) {
			return SharedFile{}, ErrNoRecord
		} else {
			return SharedFile{}, err
		}
	}

	return s, nil
}

func (m *SharedFileModel) Latest() ([]SharedFile, error) {

	stmt := `SELECT Id, DocName, RecipientName, SenderName, CreatedAt, 
       SenderEmail, RecipientEmail FROM files WHERE Expires > UTC_TIMESTAMP() ORDER BY id DESC LIMIT 10`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var sharedFiles []SharedFile

	for rows.Next() {
		var s SharedFile
		err = rows.Scan(&s.Id, &s.DocName, &s.RecipientName, &s.SenderName, &s.CreatedAt,
			&s.SenderEmail, &s.RecipientEmail)
		if err != nil {
			return nil, err
		}

		sharedFiles = append(sharedFiles, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sharedFiles, nil
}
