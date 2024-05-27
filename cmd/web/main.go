package main

import (
	"database/sql"
	"flag"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	//Internal
	"fileshare/internal/models"

	//External
	"github.com/go-playground/form/v4"
	_ "github.com/go-sql-driver/mysql"
)

type application struct {
	logger        *slog.Logger
	sharedFile    *models.SharedFileModel
	templateCache map[string]*template.Template
	formDecoder   *form.Decoder
}

func main() {

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}))

	//Get the DB Details from the .env file, !TODO: change to OS Vars in prod
	dbPass, dbUser, dbName, err := readFileEnvs(".env")
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	addr := flag.String("addr", ":4000", "HTTP network address")
	dsn := flag.String("dsn", dbUser+":"+dbPass+"@/"+dbName+"?parseTime=true", "MySQL data source name")

	flag.Parse()

	db, err := openDB(*dsn)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	defer db.Close()

	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	formDecoder := form.NewDecoder()

	app := &application{
		logger:        logger,
		sharedFile:    &models.SharedFileModel{DB: db},
		templateCache: templateCache,
		formDecoder:   formDecoder,
	}

	logger.Info("starting server on %s", *addr)

	err = http.ListenAndServe(*addr, app.routes())
	logger.Error(err.Error())
	os.Exit(1)
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func readFileEnvs(fileName string) (dbPass string, dbUser string, dbName string, err error) {

	file, err := os.Open(fileName)
	if err != nil {
		return "", "", "", err
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return "", "", "", err
	}

	text := string(data)

	dbName = getVariable(text, "DB_DATABASE")
	dbPass = getVariable(text, "DB_PASSWORD")
	dbUser = getVariable(text, "DB_USERNAME")

	return dbPass, dbUser, dbName, nil
}

func getVariable(text string, key string) string {

	lines := strings.Split(text, "\n")

	for _, line := range lines {
		if strings.Contains(line, key) {
			// Split the line into key-value pairs
			parts := strings.Split(line, "=")

			// Get the value of the variable
			return parts[1]
		}

	}
	return ""
}
