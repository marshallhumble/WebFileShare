package models

import (
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/smtp"
	"os"
)

type ServerConfigInterface interface {
	GetConfig() (ServerConfig, error)
	SendMail(rName, sName, rEmail, sEmail, fName string) error
}

type ServerConfig struct {
	mailServer   string
	mailUsername string
	mailPassword string
	mailPort     int
}

type ServerConfigModel struct {
	DB *sql.DB
}

func (m *ServerConfigModel) GetConfig() (ServerConfig, error) {
	stmt := `SELECT mail_server, mail_username, mail_password, mail_port FROM config`

	var c ServerConfig

	err := m.DB.QueryRow(stmt).Scan(&c.mailServer, &c.mailUsername, &c.mailPassword, &c.mailPort)

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

func (m *ServerConfigModel) SendMail(rName, sName, rEmail, sEmail, fName string) error {

	s, err := m.GetConfig()
	if err != nil {
		return err
	}

	delimiter := "**=myohmy689407924327"

	tlsConfig := tls.Config{
		ServerName:         s.mailServer,
		InsecureSkipVerify: true,
	}

	conn, connErr := tls.Dial("tcp", fmt.Sprintf("%s:%d", s.mailServer, s.mailPort), &tlsConfig)
	if connErr != nil {
		return connErr
	}

	defer conn.Close()

	client, clientErr := smtp.NewClient(conn, s.mailServer)
	if clientErr != nil {
		return clientErr
	}
	defer client.Close()

	auth := smtp.PlainAuth("", s.mailUsername, s.mailPassword, s.mailServer)

	if err := client.Auth(auth); err != nil {
		return err
	}

	if err := client.Mail(sEmail); err != nil {
		log.Panic(err)
	}

	writer, writerErr := client.Data()
	if writerErr != nil {
		log.Panic(writerErr)
	}

	//basic email headers
	sampleMsg := fmt.Sprintf("From: %s\r\n", sEmail)
	sampleMsg += fmt.Sprintf("To: %s\r\n", rEmail)

	sampleMsg += fmt.Sprintf("Subject: %s has sent you an attachment!\r\n", sName)

	//Mark content to accept multiple contents
	sampleMsg += "MIME-Version: 1.0\r\n"
	sampleMsg += fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", delimiter)

	//place HTML message
	sampleMsg += fmt.Sprintf("\r\n--%s\r\n", delimiter)
	sampleMsg += "Content-Type: text/html; charset=\"utf-8\"\r\n"
	sampleMsg += "Content-Transfer-Encoding: 7bit\r\n"
	sampleMsg += fmt.Sprintf("\r\n%s", "<html><body><h1>Hi There</h1>"+
		"<p>This is an email to let you know you recieved a file /p></body></html>\r\n")

	//place file
	log.Println("Put file attachment")
	sampleMsg += fmt.Sprintf("\r\n--%s\r\n", delimiter)
	sampleMsg += "Content-Type: text/plain; charset=\"utf-8\"\r\n"
	sampleMsg += "Content-Transfer-Encoding: base64\r\n"
	sampleMsg += "Content-Disposition: attachment;filename=\"" + fName + "\"\r\n"
	//read file
	rawFile, fileErr := os.ReadFile("uploads/" + fName)
	if fileErr != nil {
		return fileErr
	}
	sampleMsg += "\r\n" + base64.StdEncoding.EncodeToString(rawFile)

	//write into email client stream writter
	log.Println("Write content into client writter I/O")
	if _, err := writer.Write([]byte(sampleMsg)); err != nil {
		return err
	}

	if closeErr := writer.Close(); closeErr != nil {
		return closeErr
	}

	client.Quit()

	return nil
}
