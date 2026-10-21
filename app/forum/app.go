package forum

import (
	"bytes"
	"database/sql"
	"errors"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"
)

type App struct {
	store     *Store
	templates *template.Template
	logger    *log.Logger
	secure    bool
	secret    []byte
	now       func() time.Time
	mux       *http.ServeMux
}

func New(store *Store, templatesPath, staticPath string, secure bool, logger *log.Logger) (*App, error) {
	templates, err := template.ParseGlob(filepath.Join(templatesPath, "*.html"))
	if err != nil {
		return nil, err
	}
	secret, err := randomToken()
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = log.Default()
	}
	app := &App{store: store, templates: templates, logger: logger, secure: secure, secret: []byte(secret), now: time.Now, mux: http.NewServeMux()}
	app.mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(staticPath))))
	for _, path := range []string{"/", "/admin", "/registration", "/reg", "/login", "/log", "/logout", "/profile", "/createdposts/", "/ratedposts/", "/thread/", "/newthread", "/deletethread", "/addpost", "/newpost", "/deletepost", "/post/", "/likepost/", "/dislikepost/", "/newcomment", "/deletecomment", "/likecomm/", "/dislikecomm/"} {
		app.mux.HandleFunc(path, app.route)
	}
	return app, nil
}
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("Cache-Control", "no-store")
	if a.secure {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	}
	if len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/css/" {
		a.mux.ServeHTTP(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", 405)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	request, err := a.requestState(w, r)
	if err != nil {
		a.failure(w, err)
		return
	}
	a.mux.ServeHTTP(w, request)
}
func (a *App) render(w http.ResponseWriter, r *http.Request, name string, view View) {
	st := state(r)
	view.User = st.User
	view.Name = st.User.Name
	view.Email = st.User.Email
	view.UserID = st.User.ID
	view.CSRF = st.CSRF
	var buffer bytes.Buffer
	if err := a.templates.ExecuteTemplate(&buffer, name, view); err != nil {
		a.failure(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buffer.Bytes())
}
func (a *App) failure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		http.Error(w, "Forbidden", 403)
	case errors.Is(err, sql.ErrNoRows):
		http.Error(w, "Not found", 404)
	case errors.Is(err, ErrConflict):
		http.Error(w, "Already exists", 409)
	case constraint(err):
		http.Error(w, "Invalid or conflicting reference", 400)
	default:
		a.logger.Print("request failed: ", err)
		http.Error(w, "Internal server error", 500)
	}
}
func id(value string) (int64, error) {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 {
		return 0, errors.New("invalid ID")
	}
	return n, nil
}
func (a *App) mutation(w http.ResponseWriter, r *http.Request, admin, authenticated bool) bool {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", 405)
		return false
	}
	user := state(r).User
	if authenticated && user.ID == 0 {
		http.Error(w, "Authentication required", 401)
		return false
	}
	if admin && user.Role != "admin" {
		http.Error(w, "Forbidden", 403)
		return false
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", 400)
		return false
	}
	if !validCSRF(state(r).CSRF, r.PostForm.Get("csrf")) {
		http.Error(w, "Invalid CSRF token", 403)
		return false
	}
	return true
}
