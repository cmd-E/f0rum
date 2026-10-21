package forum

import (
	"context"
	"html/template"
	"io"
	"log"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAnonymousAdministrativeRoutes(t *testing.T) {
	app, err := New(nil, filepath.Join("..", "..", "templates"), "", false, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/deletepost", "/deletecomment", "/deletethread", "/newthread", "/newpost", "/newcomment"} {
		request := httptest.NewRequest("POST", path, strings.NewReader(url.Values{"postID": {"0 OR 1=1"}}.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		app.ServeHTTP(response, request)
		if response.Code != 401 {
			t.Errorf("%s: got %d", path, response.Code)
		}
	}
}
func TestCSRFAndPreview(t *testing.T) {
	app := &App{secret: []byte("test"), now: func() time.Time { return time.Unix(10000, 0) }, templates: template.New("test")}
	payload := "random.11000"
	token := payload + "." + app.sign("preauth:"+payload)
	if !app.validPreauth(token) || app.validPreauth(token+"x") {
		t.Fatal("signature check")
	}
	if validCSRF("", "") || validCSRF(token, token+"x") || !validCSRF(token, token) {
		t.Fatal("CSRF comparison")
	}
	for _, value := range []string{"", "1 OR 1=1", "0", "-1", "9223372036854775808", "1/2"} {
		if _, err := id(value); err == nil {
			t.Errorf("accepted ID %q", value)
		}
	}
	for _, length := range []int{0, 89, 90, 91, 99, 100} {
		text := strings.Repeat("😀", length)
		result := Preview(text)
		if length > 90 && len([]rune(result)) != 93 {
			t.Fatal("preview length")
		}
	}
}

func TestDatabaseURI(t *testing.T) {
	absolute, err := filepath.Abs(filepath.Join("data", "space name.db"))
	if err != nil {
		t.Fatal(err)
	}
	uri := databaseURI(absolute)
	if uri.Host != "" || !strings.HasPrefix(uri.String(), "file:///") || strings.Contains(uri.String(), " ") {
		t.Fatalf("invalid local database URI: %s", uri.String())
	}
}

func TestExistingTemplatesRenderWithTrustedIdentityAndCSRF(t *testing.T) {
	app, err := New(nil, filepath.Join("..", "..", "templates"), "", false, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request = request.WithContext(context.WithValue(request.Context(), stateKey{}, requestState{
		User: User{ID: 1, Name: "user", Email: "user@example.invalid", Role: "admin"}, CSRF: "test-csrf",
	}))
	post := Post{ID: 1, Title: "title", Content: "<script>unsafe</script>", Author: "user", AuthorID: 1}
	for _, name := range []string{"index.html", "admin.html", "registration.html", "login.html", "addpost.html", "profile.html", "createdposts.html", "likedposts.html", "thread.html", "post.html"} {
		view := View{Threads: []Thread{{ID: 1, Title: "thread"}}, Posts: []Post{post}, Post: []Post{post}, Comments: []Comment{{ID: 1, PostID: 1, Username: "user", Content: "comment"}}}
		if name == "post.html" {
			view.Post = post
		}
		response := httptest.NewRecorder()
		app.render(response, request, name, view)
		if response.Code != 200 {
			t.Fatalf("%s failed to render: %s", name, response.Body.String())
		}
		body := response.Body.String()
		if strings.Contains(body, "<script>unsafe</script>") {
			t.Fatalf("%s emitted unescaped content", name)
		}
		if strings.Contains(body, "<form") && !strings.Contains(body, `name="csrf" value="test-csrf"`) {
			t.Fatalf("%s omitted CSRF token", name)
		}
	}
}
