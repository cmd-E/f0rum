//go:build cgo

package forum

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func guestForm(t *testing.T, a *App, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	page := httptest.NewRecorder()
	a.ServeHTTP(page, httptest.NewRequest("GET", "/login", nil))
	var token string
	request := httptest.NewRequest("POST", path, nil)
	for _, cookie := range page.Result().Cookies() {
		request.AddCookie(cookie)
		if cookie.Name == "csrf" {
			token = cookie.Value
		}
	}
	if token == "" {
		t.Fatal("pre-authentication CSRF cookie missing")
	}
	form.Set("csrf", token)
	request = httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	for _, cookie := range page.Result().Cookies() {
		request.AddCookie(cookie)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, request)
	return w
}

func TestRegistrationAndLoginHTTPFlows(t *testing.T) {
	a, _, user := fixture(t)
	for _, form := range []url.Values{
		{"username": {"new"}, "email": {"invalid"}, "password": {"pw"}, "password_repeat": {"pw"}},
		{"username": {"new"}, "email": {"new@example.invalid"}, "password": {"pw"}, "password_repeat": {"different"}},
		{"username": {"new"}, "email": {"new@example.invalid"}, "password": {strings.Repeat("x", 73)}, "password_repeat": {strings.Repeat("x", 73)}},
	} {
		w := guestForm(t, a, "/reg", form)
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
	}
	var count int
	if err := a.store.DB.QueryRow("SELECT count(*) FROM users").Scan(&count); err != nil || count != 2 {
		t.Fatal("invalid registration persisted", count, err)
	}
	form := url.Values{"username": {"new"}, "email": {"new@example.invalid"}, "password": {"secret"}, "password_repeat": {"secret"}, "role": {"admin"}}
	if w := guestForm(t, a, "/reg", form); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	var role string
	if err := a.store.DB.QueryRow("SELECT role FROM users WHERE username='new'").Scan(&role); err != nil || role != "user" {
		t.Fatal("registration forged a role", role, err)
	}
	if w := guestForm(t, a, "/reg", form); w.Code != 200 {
		t.Fatal("duplicate registration", w.Code)
	}
	for _, password := range []string{"wrong", strings.Repeat("x", 73)} {
		w := guestForm(t, a, "/log", url.Values{"username": {user.Name}, "password": {password}})
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
	}
	w := guestForm(t, a, "/log", url.Values{"username": {user.Name}, "password": {"dummy-password"}})
	if w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestMutationHTTPValidationAndCommentVoting(t *testing.T) {
	a, admin, user := fixture(t)
	if w := send(t, a, admin, "/newthread", url.Values{"threadTitle": {""}, "threadDescription": {"description"}}); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w := send(t, a, admin, "/newthread", url.Values{"threadTitle": {"thread"}, "threadDescription": {"description"}}); w.Code != 303 {
		t.Fatal(w.Code)
	}
	for _, form := range []url.Values{
		{"postTitle": {"title"}, "postContent": {"body"}},
		{"postTitle": {"title"}, "postContent": {"body"}, "category": {"bad"}},
	} {
		if w := send(t, a, user, "/newpost", form); w.Code != 400 {
			t.Fatal(w.Code)
		}
	}
	if w := send(t, a, user, "/newpost", url.Values{"postTitle": {"title"}, "postContent": {"body"}, "category": {"1", "1"}}); w.Code != 303 {
		t.Fatal(w.Code)
	}
	if w := send(t, a, user, "/newcomment", url.Values{"postID": {"bad"}, "content": {"body"}}); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w := send(t, a, user, "/newcomment", url.Values{"postID": {"1"}, "content": {"body"}}); w.Code != 303 {
		t.Fatal(w.Code)
	}
	for _, path := range []string{"/likepost/1", "/dislikepost/1", "/likecomm/1", "/dislikecomm/1"} {
		if w := send(t, a, user, path, url.Values{"postID": {"999"}}); w.Code != 303 || w.Header().Get("Location") != "/post/1" {
			t.Fatal("untrusted redirect", path, w.Code, w.Header().Get("Location"))
		}
	}
	if w := send(t, a, user, "/likepost/invalid", url.Values{}); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w := send(t, a, admin, "/deletecomment", url.Values{"commentID": {"1"}}); w.Code != 303 {
		t.Fatal(w.Code)
	}
	if w := send(t, a, admin, "/deletethread", url.Values{"threadID": {"1"}}); w.Code != 303 {
		t.Fatal(w.Code)
	}
}
