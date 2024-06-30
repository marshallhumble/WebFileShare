package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	//Internal
	"fileshare/internal/models"
	"fileshare/internal/validator"

	//External
	"github.com/google/safeopen"
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

type userSignupForm struct {
	Id                  int    `form:"-"`
	Name                string `form:"name"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	Admin               bool   `form:"-"`
	validator.Validator `form:"-"`
}

type userLoginForm struct {
	Email               string `form:"email"`
	Password            string `form:"password"`
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
	app.render(w, r, http.StatusOK, "home.gohtml", data)
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
	app.render(w, r, http.StatusOK, "view.gohtml", data)
}

func (app *application) fileCreate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	data.Form = fileCreateForm{
		Expires: 365,
	}
	app.render(w, r, http.StatusOK, "create.gohtml", data)
}

func (app *application) fileCreatePost(w http.ResponseWriter, r *http.Request) {
	var form fileCreateForm

	if err := app.decodePostForm(r, &form); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	file, fHeader, err := r.FormFile("uploadFile")
	if err != nil {
		app.logger.Error("Handler Error: ", err)
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

	//If there are no errors let's copy the file
	if fHeader.Size > 0 {
		f, _ := safeopen.CreateAt("./uploads/", fHeader.Filename)
		defer f.Close()
		//io.Copy(f, file)
	}

	id, err := app.sharedFile.Insert(fHeader.Filename, form.RecipientUserName, form.SenderUserName, form.SenderEmail,
		form.RecipientEmail, form.Expires)

	if err != nil {
		app.serverError(w, r, err)
		return
	}

	//Let's send some mail
	if err = app.config.SendMail(form.RecipientUserName, form.SenderUserName, form.RecipientEmail,
		form.SenderEmail, fHeader.Filename); err != nil {
		app.serverError(w, r, err)
	}

	app.sessionManager.Put(r.Context(), "flash", "File successfully uploaded!")

	http.Redirect(w, r, fmt.Sprintf("/files/view/%d", id), http.StatusSeeOther)
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userSignupForm{}
	app.render(w, r, http.StatusOK, "signup.gohtml", data)
}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	var form userSignupForm

	if err := app.decodePostForm(r, &form); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX),
		"email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
	form.CheckField(validator.MinChars(form.Password, 8), "password",
		"This field must be at least 8 characters long")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "signup.gohtml", data)
		return
	}

	// Try to create a new user record in the database. If the email already
	// exists then add an error message to the form and re-display it.
	if err := app.users.Insert(form.Name, form.Email, form.Password); err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.AddFieldError("email", "Email address is already in use")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, "signup.gohtml", data)
		} else {
			app.serverError(w, r, err)
		}

		return
	}

	// Otherwise add a confirmation flash message to the session confirming that
	// their signup worked.
	app.sessionManager.Put(r.Context(), "flash", "Your signup was successful. Please log in.")

	// And redirect the user to the login page.
	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}
	app.render(w, r, http.StatusOK, "login.gohtml", data)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	// Decode the form data into the userLoginForm struct.
	var form userLoginForm

	if err := app.decodePostForm(r, &form); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Do some validation checks on the form. We check that both email and
	// password are provided, and also check the format of the email address as
	// a UX-nicety (in case the user makes a typo).
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "login.gohtml", data)
		return
	}

	// Check whether the credentials are valid. If they're not, add a generic
	// non-field error message and re-display the login page.
	id, err := app.users.Authenticate(form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, "login.gohtml", data)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	// Use the RenewToken() method on the current session to change the session
	// ID. It's good practice to generate a new session ID when the
	// authentication state or privilege levels changes for the user (e.g. login
	// and logout operations). -- OWASP Session Fixation Mitigation
	if err = app.sessionManager.RenewToken(r.Context()); err != nil {
		app.serverError(w, r, err)
		return
	}

	// Add the ID of the current user to the session, so that they are now
	// 'logged in'.
	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)

	// Redirect the user to the create snippet page.
	http.Redirect(w, r, "/files/create", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	// Use the RenewToken() method on the current session to change the session
	// ID again.
	if err := app.sessionManager.RenewToken(r.Context()); err != nil {
		app.serverError(w, r, err)
		return
	}

	// Remove the authenticatedUserID from the session data so that the user is
	// 'logged out'.
	app.sessionManager.Remove(r.Context(), "authenticatedUserID")

	// Add a flash message to the session to confirm to the user that they've been
	// logged out.
	app.sessionManager.Put(r.Context(), "flash", "You've been logged out successfully!")

	// Redirect the user to the application home page.
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func (app *application) getAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := app.users.GetAllUsers()

	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := app.newTemplateData(r)
	data.Users = users

	// Use the new render helper.
	app.render(w, r, http.StatusOK, "users.gohtml", data)

}

func (app *application) editUser(w http.ResponseWriter, r *http.Request) {

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	user, err := app.users.Get(id)
	fmt.Println(user)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := app.newTemplateData(r)
	data.User = user

	app.render(w, r, http.StatusOK, "user_edit.gohtml", data)

}

func (app *application) editUserPost(w http.ResponseWriter, r *http.Request) {
	var form userSignupForm

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	if err = app.decodePostForm(r, &form); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	/*
		form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")
		form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
		form.CheckField(validator.Matches(form.Email, validator.EmailRX),
			"email", "This field must be a valid email address")
		form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
		form.CheckField(validator.MinChars(form.Password, 8), "password",
			"This field must be at least 8 characters long")

		if !form.Valid() {
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, "user_edit.gohtml", data)
			return
		}
	*/

	user, err := app.users.UpdateUser(id, form.Name, form.Email, form.Password, form.Admin)

	data := app.newTemplateData(r)
	data.User = user

	app.render(w, r, http.StatusOK, "user_edit.gohtml", data)
}
