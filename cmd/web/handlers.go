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
	"fileshare/internal/validator"
)

type fileCreateForm struct {
	DocName             string `form:"-"`
	RecipientUserName   string `form:"recipientName"`
	RecipientEmail      string `form:"recipientEmail"`
	SenderUserName      string `form:"senderName"`
	SenderEmail         string `form:"senderEmail"`
	Expires             int    `form:"expires"`
	validator.Validator `form:"-"`
}

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

	data.Form = fileCreateForm{
		Expires: 365,
	}
	app.render(w, r, http.StatusOK, "create.tmpl", data)
}

func (app *application) fileCreatePost(w http.ResponseWriter, r *http.Request) {
	var form fileCreateForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	file, fHeader, err := r.FormFile("uploadFile")
	if err != nil {
		app.logger.Error("Handler Error: ", err)
		app.clientError(w, http.StatusBadRequest)
	}

	defer file.Close()

	form.CheckField(validator.NotBlank(form.RecipientUserName),
		"recipientName", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.RecipientEmail),
		"recipientEmail", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.SenderUserName),
		"senderName", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.SenderEmail),
		"senderEmail", "This field cannot be blank")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "create.tmpl", data)
		return
	}

	//If there are no errors let's copy the file
	f, err := os.OpenFile("./uploads/"+fHeader.Filename, os.O_WRONLY|os.O_CREATE, 0666)

	defer f.Close()
	io.Copy(f, file)

	id, err := app.sharedFile.Insert(fHeader.Filename, form.RecipientUserName, form.SenderUserName, form.Expires,
		form.SenderEmail, form.RecipientEmail)

	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "File successfully uploaded!")

	http.Redirect(w, r, fmt.Sprintf("/files/view/%d", id), http.StatusSeeOther)
}
