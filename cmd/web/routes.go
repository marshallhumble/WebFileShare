package main

import (
	"net/http"
	"path/filepath"

	//External
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	//Create file system for static files
	fileServer := http.FileServer(neuteredFileSystem{http.Dir("./ui/static")})
	mux.Handle("/static", http.NotFoundHandler())
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	//Create filesystem for uploaded files
	//uploadedFiles := http.FileServer(neuteredFileSystem{http.Dir("./uploads")})
	//mux.Handle("/uploads", http.NotFoundHandler())
	//mux.Handle("POST /uploads/", http.StripPrefix("/uploads", uploadedFiles))

	//All other routes
	mux.HandleFunc("GET /{$}", app.home)
	mux.HandleFunc("GET /files/view/{id}", app.fileView)
	mux.HandleFunc("GET /files/create", app.fileCreate)
	mux.HandleFunc("POST /files/create", app.fileCreatePost)

	standard := alice.New(app.recoverPanic, app.logRequest, commonHeaders)

	return standard.Then(mux)
}

type neuteredFileSystem struct {
	fs http.FileSystem
}

func (nfs neuteredFileSystem) Open(path string) (http.File, error) {
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if s.IsDir() {
		index := filepath.Join(path, "index.html")
		if _, err := nfs.fs.Open(index); err != nil {
			closeErr := f.Close()
			if closeErr != nil {
				return nil, closeErr
			}

			return nil, err
		}
	}

	return f, nil
}
