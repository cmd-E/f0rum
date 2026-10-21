//go:build cgo

package forum

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func sessionCookie(t *testing.T, a *App, user User) *http.Cookie {
	t.Helper()
	w := httptest.NewRecorder()
	if err := a.startSession(w, user.ID); err != nil {
		t.Fatal(err)
	}
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "session" {
			return cookie
		}
	}
	t.Fatal("session cookie missing")
	return nil
}

func TestAuthenticationAndSessionRotation(t *testing.T) {
	a, _, user := fixture(t)
	if got, err := a.store.Authenticate(user.Name, "dummy-password"); err != nil || got != user.ID {
		t.Fatal(got, err)
	}
	for _, credentials := range [][2]string{{user.Name, "wrong"}, {"missing", "dummy-password"}} {
		if _, err := a.store.Authenticate(credentials[0], credentials[1]); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("credentials accepted: %v", err)
		}
	}
	old := sessionCookie(t, a, user)
	current := sessionCookie(t, a, user)
	if old.Value == current.Value {
		t.Fatal("token reused")
	}
	var stored string
	if err := a.store.DB.QueryRow("SELECT token_hash FROM sessions WHERE user_id=?", user.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == current.Value || stored != hash(current.Value) {
		t.Fatal("raw token persisted")
	}
	for _, item := range []struct {
		cookie *http.Cookie
		code   int
	}{{old, 303}, {current, 200}, {&http.Cookie{Name: "session", Value: "forged"}, 303}} {
		r := httptest.NewRequest("GET", "/profile", nil)
		r.AddCookie(item.cookie)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		if w.Code != item.code {
			t.Fatalf("session code: %d", w.Code)
		}
	}
}

func TestCSRFFailureAndLogoutRevokeSession(t *testing.T) {
	a, _, user := fixture(t)
	cookie := sessionCookie(t, a, user)
	for _, token := range []string{"", "forged"} {
		r := httptest.NewRequest("POST", "/newpost", strings.NewReader(url.Values{"csrf": {token}, "postTitle": {"title"}, "postContent": {"body"}, "category": {"1"}}.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal(w.Code)
		}
	}
	r := httptest.NewRequest("POST", "/logout", strings.NewReader(url.Values{"csrf": {a.sign("csrf:" + cookie.Value)}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != 303 || w.Header().Get("Location") != "/login" {
		t.Fatal(w.Code)
	}
	var count int
	if err := a.store.DB.QueryRow("SELECT count(*) FROM sessions").Scan(&count); err != nil || count != 0 {
		t.Fatal(count, err)
	}
}

func TestStoreQueriesVotesAndAuthorization(t *testing.T) {
	a, admin, user := fixture(t)
	s := a.store
	if err := s.AddThread(user, "forbidden", "body"); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
	if err := s.AddPost(User{}, []int64{1}, "bad", "bad"); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
	if err := s.AddComment(User{}, 1, "bad"); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
	if err := s.Vote(User{}, false, 1, false); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
	if err := s.AddThread(admin, "thread", "description"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddPost(user, []int64{1}, "<script>title</script>", "body"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddComment(user, 1, "<script>comment</script>"); err != nil {
		t.Fatal(err)
	}
	if err := s.Vote(user, true, 1, true); err != nil {
		t.Fatal(err)
	}
	if err := s.Vote(user, true, 1, false); err != nil {
		t.Fatal(err)
	}
	comments, err := s.Comments(1)
	if err != nil || len(comments) != 1 || comments[0].Dislikes != 1 || comments[0].Likes != 0 {
		t.Fatal(comments, err)
	}
	if err := s.Vote(user, false, 1, true); err != nil {
		t.Fatal(err)
	}
	for _, filter := range []string{"thread", "author", "rated", "id"} {
		posts, err := s.Posts(filter, 1)
		if filter == "author" || filter == "rated" {
			posts, err = s.Posts(filter, user.ID)
		}
		if err != nil || len(posts) != 1 || posts[0].AuthorID != user.ID {
			t.Fatal(filter, posts, err)
		}
	}
	if _, err := s.Posts("injected SQL", 1); err == nil {
		t.Fatal("unknown filter accepted")
	}
	threads, err := s.Threads()
	if err != nil || len(threads) != 1 || threads[0].PostsCount != 1 {
		t.Fatal(threads, err)
	}
	for _, path := range []string{"/", "/post/1", "/thread/1", "/createdposts/2", "/ratedposts/2", "/missing", "/post/invalid", "/post/999"} {
		w := httptest.NewRecorder()
		a.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		want := 200
		if path == "/missing" || path == "/post/999" {
			want = 404
		}
		if path == "/post/invalid" {
			want = 400
		}
		if w.Code != want || strings.Contains(w.Body.String(), "<script>title</script>") || strings.Contains(w.Body.String(), "<script>comment</script>") {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	if err := s.Delete(user, "post", 1); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
	if err := s.Delete(admin, "unknown", 1); err == nil {
		t.Fatal("unknown deletion accepted")
	}
	if err := s.Delete(admin, "comment", 1); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(admin, "thread", 1); err != nil {
		t.Fatal(err)
	}
}

func TestInitializeIsIdempotentAndRejectsFutureSchema(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	for i := 0; i < 2; i++ {
		if err := s.Initialize(); err != nil {
			t.Fatal(err)
		}
	}
	if version, err := s.Version(); err != nil || version != 1 {
		t.Fatal(version, err)
	}
	if err := s.Migrate("unused", 999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing admin: %v", err)
	}
	if _, err := s.DB.Exec("PRAGMA user_version=99"); err != nil {
		t.Fatal(err)
	}
	if err := s.Initialize(); err == nil {
		t.Fatal("future schema accepted")
	}
	if err := s.Migrate("unused", 0); err == nil {
		t.Fatal("future migration accepted")
	}
}
