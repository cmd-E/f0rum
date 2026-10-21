package forum

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPreauthTokenBoundaries(t *testing.T) {
	a := &App{secret: []byte("secret"), now: func() time.Time { return time.Unix(10000, 0) }}
	for _, seconds := range []int64{9999, 10000, 10001, 13600, 13601} {
		payload := fmt.Sprintf("random.%d", seconds)
		token := payload + "." + a.sign("preauth:"+payload)
		if got, want := a.validPreauth(token), seconds > 10000 && seconds <= 13600; got != want {
			t.Errorf("expiry %d: got %t want %t", seconds, got, want)
		}
	}
	for _, token := range []string{"", "random", "a.b.c.d", "a.invalid.sig", "a.11000.sig"} {
		if a.validPreauth(token) {
			t.Fatalf("accepted %q", token)
		}
	}
}

func TestMutationRequiresMethodIdentityRoleAndCSRF(t *testing.T) {
	a := &App{}
	cases := []struct {
		name, method, role, csrf string
		user                     int64
		admin, auth              bool
		code                     int
	}{
		{"method", "GET", "admin", "good", 1, true, true, 405},
		{"anonymous", "POST", "", "good", 0, false, true, 401},
		{"role", "POST", "user", "good", 1, true, true, 403},
		{"csrf", "POST", "admin", "bad", 1, true, true, 403},
		{"valid", "POST", "admin", "good", 1, true, true, 200},
		{"public", "POST", "", "good", 0, false, false, 200},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			r := httptest.NewRequest(item.method, "/newthread", strings.NewReader(url.Values{"csrf": {item.csrf}}.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r = r.WithContext(context.WithValue(r.Context(), stateKey{}, requestState{User: User{ID: item.user, Role: item.role}, CSRF: "good"}))
			w := httptest.NewRecorder()
			allowed := a.mutation(w, r, item.admin, item.auth)
			if allowed != (item.code == 200) || w.Code != item.code {
				t.Fatalf("allowed=%v code=%d", allowed, w.Code)
			}
		})
	}
}

func TestSecureCookiesAndHeaders(t *testing.T) {
	a := &App{secure: true}
	w := httptest.NewRecorder()
	a.cookie(w, "session", "opaque", 86400)
	c := w.Result().Cookies()[0]
	if !c.Secure || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.MaxAge != 86400 {
		t.Fatalf("unsafe cookie: %+v", c)
	}
	r := httptest.NewRequest("DELETE", "/", nil)
	a.ServeHTTP(w, r)
	if w.Code != 405 || w.Header().Get("Strict-Transport-Security") == "" || w.Header().Get("X-Frame-Options") != "DENY" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing security policy")
	}
}

func TestPreviewBoundariesAndIDs(t *testing.T) {
	for _, length := range []int{0, 1, 89, 90, 91} {
		text := strings.Repeat("ж", length)
		want := text
		if length > 90 {
			want = strings.Repeat("ж", 90) + "..."
		}
		if got := Preview(text); got != want {
			t.Errorf("length %d: %q", length, got)
		}
	}
	if got, err := id("9223372036854775807"); err != nil || got != 9223372036854775807 {
		t.Fatal(got, err)
	}
}
