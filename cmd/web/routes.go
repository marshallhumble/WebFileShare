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
	//mux.Handle("GET /static/", http.FileServerFS(ui.Files))

	//PING! PONG! used for testing
	mux.HandleFunc("GET /ping", ping)

	//dynamic middleware route
	dynamic := alice.New(app.sessionManager.LoadAndSave, noSurf, app.authenticate)

	//Make Alice Login protected route
	protected := dynamic.Append(app.requireAuthentication)

	//Make Alice Admin Only route
	admin := dynamic.Append(app.requireAdmin)

	//Default route
	mux.Handle("GET /{$}", dynamic.ThenFunc(app.home))

	//Protected File Create/View Routes
	mux.Handle("GET /files/view/{id}", dynamic.ThenFunc(app.fileView))
	mux.Handle("GET /files/create", protected.ThenFunc(app.fileCreate))
	mux.Handle("POST /files/create", protected.ThenFunc(app.fileCreatePost))
	mux.Handle("GET /files/download/{file}", protected.ThenFunc(app.fileDownload))
	mux.Handle("GET /files/delete/{id}", protected.ThenFunc(app.fileDelete))

	//Protected User Routes
	mux.Handle("GET /users/", admin.ThenFunc(app.getAllUsers))
	mux.Handle("GET /user/edit/{id}", protected.ThenFunc(app.editUser))
	mux.Handle("POST /user/edit/{id}", protected.ThenFunc(app.editUserPost))
	mux.Handle("POST /user/delete/{id}", protected.ThenFunc(app.deleteUser))
	mux.Handle("GET /user/update/", protected.ThenFunc(app.updateUser))
	mux.Handle("POST /user/update/", protected.ThenFunc(app.updateUserPost))

	//User Sign-up/Login/Logout
	mux.Handle("GET /user/signup", dynamic.ThenFunc(app.userSignup))
	mux.Handle("POST /user/signup", dynamic.ThenFunc(app.userSignupPost))
	mux.Handle("GET /user/login", dynamic.ThenFunc(app.userLogin))
	mux.Handle("POST /user/login", dynamic.ThenFunc(app.userLoginPost))
	mux.Handle("POST /user/logout", dynamic.ThenFunc(app.userLogoutPost))

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
