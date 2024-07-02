package models

import (
	"database/sql"
	"errors"
	"net/smtp"
	"strconv"
)

type ServerConfigInterface interface {
	GetConfig() (ServerConfig, error)
	SendMail(rName, sName, rEmail, sEmail, fName, password string) error
}

type ServerConfig struct {
	mailServer   string
	mailUsername string
	mailPassword string
	mailPort     int
	serverName   string
}

type ServerConfigModel struct {
	DB *sql.DB
}

func (m *ServerConfigModel) GetConfig() (ServerConfig, error) {
	stmt := `SELECT mail_server, mail_username, mail_password, mail_port, server_name FROM config`

	var c ServerConfig

	err := m.DB.QueryRow(stmt).Scan(&c.mailServer, &c.mailUsername, &c.mailPassword, &c.mailPort, &c.serverName)

	if err != nil {
		// If the query returns no rows, then row.Scan() will return a
		// sql.ErrNoRows error. We use the errors.Is() function check for that
		// error specifically, and return our own ErrNoRecord error
		// instead.
		if errors.Is(err, sql.ErrNoRows) {
			return ServerConfig{}, ErrNoRecord
		} else {
			return ServerConfig{}, err
		}
	}

	return c, nil
}

func (m *ServerConfigModel) SendMail(rName, sName, rEmail, sEmail, fName, password string) error {

	s, err := m.GetConfig()
	if err != nil {
		return err
	}
	server := s.mailServer + ":" + strconv.Itoa(s.mailPort)
	auth := smtp.PlainAuth("", s.mailUsername, s.mailPassword, s.mailServer)

	// Here we do it all: connect to our server, set up a message and send it

	to := []string{rEmail}

	MsgFile := []byte("To: " + rEmail + "\r\n" +
		"Subject: " + sName + " has sent you" + fName + "\r\n" +
		"\r\n" +
		rName + " go to webapp https://" + s.serverName + " for your file!\r\n")

	err = smtp.SendMail(server, auth, sEmail, to, MsgFile)
	if err != nil {
		return err
	}

	MsgPassword := []byte("To: " + rEmail + "\r\n" +
		"Subject: " + sName + " has sent you" + fName + "\r\n" +
		"\r\n" +
		rName + " here is your password " + password + "\r\n")

	err = smtp.SendMail(server, auth, sEmail, to, MsgPassword)
	if err != nil {
		return err
	}

	return nil
}
