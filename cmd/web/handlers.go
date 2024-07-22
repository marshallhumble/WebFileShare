package main

import (
	"errors"
	"fmt"
	"github.com/google/safeopen"
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

	fmt.Printf("auth: %t, admin: %t, user: %t, guest: %t\n", auth, admin, user, guest)

	if guest && !admin && !user {
		email := app.sessionManager.Get(r.Context(), "authenticatedUserEmail")
		if email == nil {
			app.clientError(w, http.StatusBadRequest)
			fmt.Printf("guest: %t", guest)
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
		fmt.Printf("admin: %t", admin)
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
		fmt.Printf("user: %t", user)
		data := app.newTemplateData(r)
		data.SharedFiles = sharedFiles

		app.render(w, r, http.StatusOK, "home.gohtml", data)
		return
	}

	if !auth {
		data := app.newTemplateData(r)
		fmt.Println("no auth")
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

	password := app.RandPasswordGen(15)

	//If there are no errors let's copy the file
	if fHeader.Size > 0 {
		f, _ := safeopen.CreateAt("./uploads/", fHeader.Filename)
		defer f.Close()
	}

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
	// Insert(name, email, password string, admin, guest bool)
	if err := app.users.Insert(form.RecipientUserName, form.RecipientEmail, password, false, true); err != nil {
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
	if err := app.users.Insert(form.Name, form.Email, form.Password, false, false); err != nil {
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

	// Add the ID & Email of the current user to the session, so that they are now
	// 'logged in'.
	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)
	app.sessionManager.Put(r.Context(), "authenticatedUserEmail", form.Email)

	// Redirect the user to the create snippet page.
	http.Redirect(w, r, "/", http.StatusSeeOther)
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
	if !app.isAuthenticated(r) {
		data := app.newTemplateData(r)
		app.render(w, r, http.StatusOK, "home.gohtml", data)
		return
	}

	users, err := app.users.GetAllUsers()

	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := app.newTemplateData(r)
	data.Users = users

	app.render(w, r, http.StatusOK, "users.gohtml", data)

}

func (app *application) editUser(w http.ResponseWriter, r *http.Request) {
	if !app.isAuthenticated(r) {
		data := app.newTemplateData(r)
		app.render(w, r, http.StatusOK, "home.gohtml", data)
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	user, err := app.users.Get(id)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := app.newTemplateData(r)
	data.User = user

	app.render(w, r, http.StatusOK, "user_edit.gohtml", data)

}

func (app *application) editUserPost(w http.ResponseWriter, r *http.Request) {
	if !app.isAuthenticated(r) {
		data := app.newTemplateData(r)
		app.render(w, r, http.StatusOK, "home.gohtml", data)
		return
	}

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

	user, err := app.users.UpdateUser(id, form.Name, form.Email, form.Password, form.Admin)

	data := app.newTemplateData(r)
	data.User = user

	app.render(w, r, http.StatusOK, "user_edit.gohtml", data)
}

func (app *application) deleteUser(w http.ResponseWriter, r *http.Request) {
	if !app.isAdmin(r) {
		data := app.newTemplateData(r)
		app.render(w, r, http.StatusOK, "home.gohtml", data)
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		http.NotFound(w, r)
	}

	if err := app.users.DeleteUser(id); err != nil {
		app.serverError(w, r, err)
	}

	app.sessionManager.Put(r.Context(), "flash", "User Deleted")
	http.Redirect(w, r, "/users/", http.StatusSeeOther)
}
