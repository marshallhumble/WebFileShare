package main

import (
	"net/http"
	"testing"

	"fileshare/internal/assert"
)

func TestPing(t *testing.T) {
	app := newTestApplication(t)

	ts := newTestServer(t, app.routes())
	defer ts.Close()

	code, _, body := ts.get(t, "/ping")

	assert.Equal(t, code, http.StatusOK)
	assert.Equal(t, body, "OK")
}

// This is broken due to not being logged in
func TestFileView(t *testing.T) {
	// Create a new instance of our application struct which uses the mocked
	// dependencies.
	app := newTestApplication(t)

	// Establish a new test server for running end-to-end tests.
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	// Set up some table-driven tests to check the responses sent by our
	// application for different URLs.
	tests := []struct {
		name     string
		urlPath  string
		wantCode int
		wantBody string
	}{
		{
			name:     "Valid ID",
			urlPath:  "/files/view/1",
			wantCode: http.StatusOK,
			wantBody: "Abar@example.com",
		},
		{
			name:     "Non-existent ID",
			urlPath:  "/files/view/2",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Negative ID",
			urlPath:  "/files/view/-1",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Decimal ID",
			urlPath:  "/files/view/1.23",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "String ID",
			urlPath:  "/files/view/foo",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Empty ID",
			urlPath:  "/files/view/",
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, body := ts.get(t, tt.urlPath)

			assert.Equal(t, code, tt.wantCode)

			if tt.wantBody != "" {
				assert.StringContains(t, body, tt.wantBody)
			}
		})
	}
}

func TestUserSignup(t *testing.T) {
	// Create the application struct containing our mocked dependencies and set
	// up the test server for running an end-to-end test.
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	// Make a GET /user/signup request and then extract the CSRF token from the
	// response body.
	_, _, body := ts.get(t, "/user/signup")
	csrfToken := extractCSRFToken(t, body)

	// Log the CSRF token value in our test output using the t.Logf() function.
	// The t.Logf() function works in the same way as fmt.Printf(), but writes
	// the provided message to the test output.
	t.Logf("CSRF token is: %q", csrfToken)

}
