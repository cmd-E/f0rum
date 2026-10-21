//go:build cgo
// +build cgo

package forum

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) (*App, User, User) {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.DB.Close() })
	if err = s.Initialize(); err != nil {
		t.Fatal(err)
	}
	aID, err := s.Register("admin", "admin@example.invalid", "dummy-password")
	if err != nil {
		t.Fatal(err)
	}
	uID, err := s.Register("user", "user@example.invalid", "dummy-password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("UPDATE users SET role='admin' WHERE id=?", aID); err != nil {
		t.Fatal(err)
	}
	admin, _ := s.User(aID)
	user, _ := s.User(uID)
	app, err := New(s, filepath.Join("..", "..", "templates"), "", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	return app, admin, user
}
func send(t *testing.T, a *App, u User, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	session := httptest.NewRecorder()
	if err := a.startSession(session, u.ID); err != nil {
		t.Fatal(err)
	}
	var cookie *http.Cookie
	for _, c := range session.Result().Cookies() {
		if c.Name == "session" {
			cookie = c
		}
	}
	form.Set("csrf", a.sign("csrf:"+cookie.Value))
	request := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	a.ServeHTTP(response, request)
	return response
}
func TestIdentityAuthorizationAndAtomicWrites(t *testing.T) {
	a, admin, user := fixture(t)
	if _, err := a.store.Register("admin", "other@example.invalid", "dummy-password"); err != ErrConflict {
		t.Fatalf("duplicate accepted: %v", err)
	}
	if err := a.store.AddThread(admin, "thread", "description"); err != nil {
		t.Fatal(err)
	}
	if response := send(t, a, user, "/deletepost", url.Values{"postID": {"1"}}); response.Code != 403 {
		t.Fatal(response.Code)
	}
	if response := send(t, a, admin, "/deletepost", url.Values{"postID": {"0 OR 1=1"}}); response.Code != 400 {
		t.Fatal(response.Code)
	}
	if response := send(t, a, user, "/newpost", url.Values{"category": {"1"}, "postTitle": {"title"}, "postContent": {"body"}, "authorID": {formatID(admin.ID)}, "postAuthor": {"admin"}}); response.Code != 303 {
		t.Fatal(response.Code, response.Body.String())
	}
	var author int64
	a.store.DB.QueryRow("SELECT authorid FROM posts").Scan(&author)
	if author != user.ID {
		t.Fatal("forged identity")
	}
	if err := a.store.AddPost(user, []int64{999}, "bad", "body"); err == nil {
		t.Fatal("invalid category accepted")
	}
	var count int
	a.store.DB.QueryRow("SELECT count(*) FROM posts").Scan(&count)
	if count != 1 {
		t.Fatal("partial post persisted")
	}
	if err := a.store.Vote(user, false, 1, false); err != nil {
		t.Fatal(err)
	}
	if err := a.store.Vote(user, false, 1, false); err != nil {
		t.Fatal(err)
	}
	a.store.DB.QueryRow("SELECT likes FROM posts WHERE id=1").Scan(&count)
	if count != 1 {
		t.Fatal("duplicate vote")
	}
	if err := a.store.AddComment(user, 1, "comment"); err != nil {
		t.Fatal(err)
	}
	if err := a.store.Delete(admin, "post", 1); err != nil {
		t.Fatal(err)
	}
	a.store.DB.QueryRow("SELECT count(*) FROM comments").Scan(&count)
	if count != 0 {
		t.Fatal("orphan comments")
	}
	// Expired server-side session must not authorize a mutation.
	recorder := httptest.NewRecorder()
	a.startSession(recorder, admin.ID)
	a.store.DB.Exec("UPDATE sessions SET expires_at=?", time.Now().Add(-time.Hour).Unix())
	request := httptest.NewRequest("POST", "/newthread", strings.NewReader("csrf=bad"))
	request.AddCookie(recorder.Result().Cookies()[0])
	response := httptest.NewRecorder()
	a.ServeHTTP(response, request)
	if response.Code != 401 {
		t.Fatal("expired session accepted")
	}
}
