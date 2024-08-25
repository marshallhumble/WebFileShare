package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	//Internal
	"fileshare/internal/models"

	//External
	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	_ "github.com/go-sql-driver/mysql"
)

type application struct {
	logger         *slog.Logger
	sharedFile     models.SharedFileModelInterface
	users          models.UserModelInterface
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
	config         models.ServerConfigInterface
}

// MaxUploadSize defines the largest file that can be uploaded in the system
const MaxUploadSize = 2024 * 2024

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

	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db)
	sessionManager.Lifetime = 12 * time.Hour
	sessionManager.IdleTimeout = 15 * time.Minute
	sessionManager.Cookie.Secure = true

	app := &application{
		logger:         logger,
		sharedFile:     &models.SharedFileModel{DB: db},
		users:          &models.UserModel{DB: db},
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
		config:         &models.ServerConfigModel{DB: db},
	}

	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
	}

	srv := &http.Server{
		Addr:         *addr,
		Handler:      app.routes(),
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
		TLSConfig:    tlsConfig,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger.Info("starting server", "addr", srv.Addr)

	err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	logger.Error(err.Error())
	os.Exit(1)
}

// openDB open the db and check if the tables exist, if not run first setup.
func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// readFileEnvs pull the database details from the .ENV file that we are using for Docker init
func readFileEnvs(fileName string) (dbPass, dbUser, dbName string, err error) {

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

// getVariable get the variables from the ENV file, right now we are assuming they look like this:
// DB_USERNAME=username
// DB_PASSWORD=password
// DB_DATABASE=db_name
func getVariable(text, key string) string {

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
