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

	//dynamic middleware route
	dynamic := alice.New(app.sessionManager.LoadAndSave)

	//All other routes
	mux.Handle("GET /{$}", dynamic.ThenFunc(app.home))
	mux.Handle("GET /files/view/{id}", dynamic.ThenFunc(app.fileView))
	mux.Handle("GET /files/create", dynamic.ThenFunc(app.fileCreate))
	mux.Handle("POST /files/create", dynamic.ThenFunc(app.fileCreatePost))

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
