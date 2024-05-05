package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	//Internal
	"fileshare/internal/models"
)

const MAX_UPLOAD_SIZE = 1024 * 1024

func (app *application) home(w http.ResponseWriter, r *http.Request) {

	sharedFiles, err := app.sharedFile.Latest()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := app.newTemplateData(r)
	data.SharedFiles = sharedFiles

	// Use the new render helper.
	app.render(w, r, http.StatusOK, "home.tmpl", data)
}

func (app *application) fileView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	sharedF, err := app.sharedFile.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	data := app.newTemplateData(r)
	data.SharedFile = sharedF

	// Use the new render helper.
	app.render(w, r, http.StatusOK, "view.tmpl", data)
}

func (app *application) fileCreate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	app.render(w, r, http.StatusOK, "create.tmpl", data)
}

func (app *application) fileCreatePost(w http.ResponseWriter, r *http.Request) {

	r.Body = http.MaxBytesReader(w, r.Body, MAX_UPLOAD_SIZE)
	if err := r.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	file, fHeader, err := r.FormFile("uploadFile")

	if err != nil {
		app.logger.Error("Handler Error: ", err)
		app.clientError(w, http.StatusBadRequest)
	}

	defer file.Close()

	f, err := os.OpenFile("./uploads/"+fHeader.Filename, os.O_WRONLY|os.O_CREATE, 0666)

	defer f.Close()
	io.Copy(f, file)

	var (
		docName           = fHeader.Filename
		recipientUserName = r.PostFormValue("recipientName")
		senderUserName    = r.PostFormValue("senderName")
		expiresAt         = 7
		senderEmail       = r.PostFormValue("senderEmail")
		recipientEmail    = r.PostFormValue("recipientEmail")
	)

	id, err := app.sharedFile.Insert(docName, recipientUserName, senderUserName, expiresAt, senderEmail,
		recipientEmail)

	if err != nil {
		app.serverError(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/files/view/%d", id), http.StatusSeeOther)
}
