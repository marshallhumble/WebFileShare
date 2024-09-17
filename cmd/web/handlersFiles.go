package main

import (
	"errors"
	"fmt"
	"github.com/google/safeopen"
	"io"
	"net/http"
	"os"
	"strconv"

	//Internal
	"fileshare/internal/models"
	"fileshare/internal/validator"
)

type fileCreateForm struct {
	DocName             string `form:"docName"`
	RecipientUserName   string `form:"recipientName"`
	RecipientEmail      string `form:"recipientEmail"`
	SenderUserName      string `form:"senderName"`
	SenderEmail         string `form:"senderEmail"`
	Expires             int    `form:"expires"`
	validator.Validator `form:"-"`
}

// home Want to show a different page for guest, admin, regular users and non-authenticated users.
// All users get authenticated, so we need to filter on guest and admin to limit views, also to not
// show duplicate home pages.
func (app *application) home(w http.ResponseWriter, r *http.Request) {

	var (
		auth  = app.isAuthenticated(r)
		admin = app.isAdmin(r)
		user  = app.isUser(r)
		guest = app.isGuest(r)
	)

	if guest && !admin && !user {
		email := app.sessionManager.Get(r.Context(), "authenticatedUserEmail")
		if email == nil {
			app.clientError(w, http.StatusBadRequest)
			return
		}

		sharedFiles, err := app.sharedFile.GetFileFromEmail(email.(string))
		if err != nil {
			app.serverError(w, r, err)
			return
		}
		data := app.newTemplateData(r)
		data.SharedFiles = sharedFiles

		app.render(w, r, http.StatusOK, "home.gohtml", data)
	}

	if admin {
		sharedFiles, err := app.sharedFile.Latest()
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		data := app.newTemplateData(r)
		data.SharedFiles = sharedFiles
		app.render(w, r, http.StatusOK, "home.gohtml", data)
	}

	if user {
		email := app.sessionManager.Get(r.Context(), "authenticatedUserEmail")
		if email == nil {
			app.clientError(w, http.StatusBadRequest)
			return
		}
		sharedFiles, err := app.sharedFile.GetCreatedFiles(email.(string))
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		data := app.newTemplateData(r)
		data.SharedFiles = sharedFiles

		app.render(w, r, http.StatusOK, "home.gohtml", data)
		return
	}

	if !auth {
		data := app.newTemplateData(r)
		app.render(w, r, http.StatusSeeOther, "home.gohtml", data)
	}

}

func (app *application) fileView(w http.ResponseWriter, r *http.Request) {
	/*if !app.isAuthenticated(r) {
		data := app.newTemplateData(r)
		app.render(w, r, http.StatusOK, "home.gohtml", data)
		return
	}*/

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

	app.render(w, r, http.StatusOK, "view.gohtml", data)
}

func (app *application) fileCreate(w http.ResponseWriter, r *http.Request) {
	if !app.isAuthenticated(r) {
		data := app.newTemplateData(r)
		app.render(w, r, http.StatusOK, "home.gohtml", data)
		return
	}

	data := app.newTemplateData(r)

	data.Form = fileCreateForm{
		Expires: 365,
	}
	app.render(w, r, http.StatusOK, "create.gohtml", data)
}

func (app *application) fileCreatePost(w http.ResponseWriter, r *http.Request) {
	if !app.isAuthenticated(r) {
		data := app.newTemplateData(r)
		app.render(w, r, http.StatusOK, "home.gohtml", data)
		return
	}

	var form fileCreateForm

	if err := app.decodePostForm(r, &form); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	file, fHeader, err := r.FormFile("uploadFile")
	if err != nil {
		app.logger.Error("Handler Error: ", err.Error(), "error")
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnsupportedMediaType, "create.gohtml", data)
	}

	defer file.Close()

	form.CheckField(validator.NotBlank(form.RecipientUserName),
		"recipientName", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.RecipientEmail),
		"recipientEmail", "This field cannot be blank")
	form.CheckField(validator.Matches(form.RecipientEmail, validator.EmailRX),
		"recipientEmail", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.SenderUserName),
		"senderName", "This field cannot be blank")
	form.CheckField(validator.Matches(form.SenderEmail, validator.EmailRX),
		"senderEmail", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.SenderEmail),
		"senderEmail", "This field cannot be blank")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "create.gohtml", data)
		return
	}

	password := app.RandPasswordGen(15)

	//If there are no errors let's copy the file
	if fHeader.Size > 0 {
		f, _ := safeopen.CreateAt("./uploads/", fHeader.Filename)
		defer f.Close()

		_, err := io.Copy(f, file)
		if err != nil {
			return
		}
	}

	//Insert(docName, senderUserName, senderEmail, recipientUserName, recipientEmail,
	//		password string, expiresAt int) (int, error)
	id, err := app.sharedFile.Insert(fHeader.Filename, form.SenderUserName, form.SenderEmail, form.RecipientUserName,
		form.RecipientEmail, password, form.Expires)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.logger.Info("File uploaded", "id: ", id)

	//Let's send some mail
	if err = app.config.SendMail(form.RecipientUserName, form.SenderUserName, form.RecipientEmail,
		form.SenderEmail, fHeader.Filename, password); err != nil {
		app.serverError(w, r, err)
	}
	app.logger.Info("Email sent! ", "email: ", form.RecipientEmail)

	// Insert(name, email, password string, admin, user, guest, disabled bool) error
	if err := app.users.Insert(form.RecipientUserName, form.RecipientEmail, password, false, false,
		true, false); err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			app.sessionManager.Put(r.Context(), "flash", "Email address is already in use, no new account created")
		} else {
			app.serverError(w, r, err)
		}
	}

	app.logger.Info("User created! ", "user: ", form.RecipientEmail)

	app.sessionManager.Put(r.Context(), "flash", "File successfully uploaded!")
	http.Redirect(w, r, fmt.Sprintf("/files/view/%d", id), http.StatusSeeOther)
}

func (app *application) fileDownload(w http.ResponseWriter, r *http.Request) {
	if !app.isAuthenticated(r) {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	file := r.PathValue("file")
	http.ServeFile(w, r, "./uploads/"+file)
}

func (app *application) fileDelete(w http.ResponseWriter, r *http.Request) {
	if !app.isAuthenticated(r) {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		http.NotFound(w, r)
	}

	if err := app.sharedFile.Remove(id); err != nil {
		app.serverError(w, r, err)
	}

	file := fmt.Sprintf("./uploads/" + r.PathValue("file"))

	if err := os.Remove(file); err != nil {
		app.logger.Info("Error removing file", "error", err)
	} else {
		app.logger.Info("File removed", "filename", r.FormValue("DocName"))
	}

	app.sessionManager.Put(r.Context(), "flash", "File successfully deleted!")
	http.Redirect(w, r, fmt.Sprintf("/"), http.StatusSeeOther)
	return

}
