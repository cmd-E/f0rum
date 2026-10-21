package forum

import (
	"database/sql"
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
)

func (a *App) route(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	user := state(r).User
	if r.Method == http.MethodPost || isMutation(path) {
		a.mutate(w, r)
		return
	}
	view := View{}
	switch {
	case path == "/":
		threads, err := a.store.Threads()
		if err != nil {
			a.failure(w, err)
			return
		}
		view.Threads = threads
		a.render(w, r, "index.html", view)
	case path == "/login":
		a.render(w, r, "login.html", view)
	case path == "/registration":
		a.render(w, r, "registration.html", view)
	case path == "/admin":
		if user.Role != "admin" {
			http.Error(w, "Forbidden", 403)
			return
		}
		a.render(w, r, "admin.html", view)
	case path == "/addpost":
		if user.ID == 0 {
			http.Redirect(w, r, "/login", 303)
			return
		}
		threads, err := a.store.Threads()
		if err != nil {
			a.failure(w, err)
			return
		}
		view.Threads = threads
		a.render(w, r, "addpost.html", view)
	case path == "/profile":
		if user.ID == 0 {
			http.Redirect(w, r, "/login", 303)
			return
		}
		err := a.store.DB.QueryRow("SELECT (SELECT count(*) FROM posts WHERE authorid=?),(SELECT count(*) FROM rtactions WHERE user_id=?)", user.ID, user.ID).Scan(&view.CreatedPostsN, &view.RatedPostsN)
		if err != nil {
			a.failure(w, err)
			return
		}
		a.render(w, r, "profile.html", view)
	case strings.HasPrefix(path, "/post/"):
		postID, err := id(strings.TrimPrefix(path, "/post/"))
		if err != nil {
			http.Error(w, "Invalid ID", 400)
			return
		}
		posts, err := a.store.Posts("id", postID)
		if err != nil {
			a.failure(w, err)
			return
		}
		if len(posts) == 0 {
			a.failure(w, sql.ErrNoRows)
			return
		}
		view.Post = posts[0]
		view.Comments, err = a.store.Comments(postID)
		if err != nil {
			a.failure(w, err)
			return
		}
		a.render(w, r, "post.html", view)
	case strings.HasPrefix(path, "/thread/") || strings.HasPrefix(path, "/createdposts/") || strings.HasPrefix(path, "/ratedposts/"):
		prefix, filter, name := "/thread/", "thread", "thread.html"
		if strings.HasPrefix(path, "/createdposts/") {
			prefix, filter, name = "/createdposts/", "author", "createdposts.html"
		}
		if strings.HasPrefix(path, "/ratedposts/") {
			prefix, filter, name = "/ratedposts/", "rated", "likedposts.html"
		}
		objectID, err := id(strings.TrimPrefix(path, prefix))
		if err != nil {
			http.Error(w, "Invalid ID", 400)
			return
		}
		posts, err := a.store.Posts(filter, objectID)
		if err != nil {
			a.failure(w, err)
			return
		}
		for i := range posts {
			posts[i].Content = Preview(posts[i].Content)
		}
		view.Posts = posts
		view.Post = posts
		a.render(w, r, name, view)
	default:
		http.NotFound(w, r)
	}
}
func isMutation(path string) bool {
	for _, value := range []string{"/reg", "/log", "/logout", "/newthread", "/deletethread", "/newpost", "/deletepost", "/newcomment", "/deletecomment"} {
		if path == value {
			return true
		}
	}
	for _, value := range []string{"/likepost/", "/dislikepost/", "/likecomm/", "/dislikecomm/"} {
		if strings.HasPrefix(path, value) {
			return true
		}
	}
	return false
}
func (a *App) mutate(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if !isMutation(path) {
		http.Error(w, "Method not allowed", 405)
		return
	}
	admin := path == "/newthread" || path == "/deletethread" || path == "/deletepost" || path == "/deletecomment"
	if !a.mutation(w, r, admin, path != "/reg" && path != "/log") {
		return
	}
	user := state(r).User
	redirect := "/"
	var err error
	switch {
	case path == "/reg":
		username, email := strings.TrimSpace(r.PostForm.Get("username")), strings.TrimSpace(r.PostForm.Get("email"))
		password := r.PostForm.Get("password")
		address, emailErr := mail.ParseAddress(email)
		if !nonempty(username) || len(username) > 64 || len(email) > 254 || emailErr != nil || address.Address != email || !nonempty(password) || len(password) > 72 || password != r.PostForm.Get("password_repeat") {
			a.render(w, r, "registration.html", View{Wrong: true})
			return
		}
		var userID int64
		userID, err = a.store.Register(username, email, password)
		if errors.Is(err, ErrConflict) {
			a.render(w, r, "registration.html", View{UserExist: true})
			return
		}
		if err == nil {
			err = a.startSession(w, userID)
		}
	case path == "/log":
		var userID int64
		password := r.PostForm.Get("password")
		if len(password) > 72 {
			a.render(w, r, "login.html", View{Wrong: true})
			return
		}
		userID, err = a.store.Authenticate(r.PostForm.Get("username"), password)
		if errors.Is(err, sql.ErrNoRows) {
			a.render(w, r, "login.html", View{Wrong: true})
			return
		}
		if err == nil {
			err = a.startSession(w, userID)
		}
	case path == "/logout":
		_, err = a.store.DB.Exec("DELETE FROM sessions WHERE token_hash=?", hash(state(r).Token))
		a.cookie(w, "session", "", -1)
		redirect = "/login"
	case path == "/newthread":
		title, description := strings.TrimSpace(r.PostForm.Get("threadTitle")), strings.TrimSpace(r.PostForm.Get("threadDescription"))
		if !nonempty(title) || len(title) > 200 || !nonempty(description) {
			http.Error(w, "Invalid thread", 400)
			return
		}
		err = a.store.AddThread(user, title, description)
		redirect = "/admin"
	case path == "/deletethread" || path == "/deletepost" || path == "/deletecomment":
		kind, field := "post", "postID"
		if path == "/deletethread" {
			kind, field = "thread", "threadID"
		}
		if path == "/deletecomment" {
			kind, field = "comment", "commentID"
		}
		objectID, parseErr := id(r.PostForm.Get(field))
		if parseErr != nil {
			http.Error(w, "Invalid ID", 400)
			return
		}
		err = a.store.Delete(user, kind, objectID)
		redirect = "/admin"
	case path == "/newpost":
		title, content := strings.TrimSpace(r.PostForm.Get("postTitle")), strings.TrimSpace(r.PostForm.Get("postContent"))
		categories := r.PostForm["category"]
		if len(categories) == 0 && r.PostForm.Get("postThreadID") != "" {
			categories = []string{r.PostForm.Get("postThreadID")}
		}
		if !nonempty(title) || len(title) > 200 || !nonempty(content) || len(content) > 100000 || len(categories) == 0 {
			http.Error(w, "Invalid post", 400)
			return
		}
		ids := []int64{}
		seen := map[int64]bool{}
		for _, value := range categories {
			n, parseErr := id(value)
			if parseErr != nil {
				http.Error(w, "Invalid category", 400)
				return
			}
			if !seen[n] {
				ids = append(ids, n)
				seen[n] = true
			}
		}
		err = a.store.AddPost(user, ids, title, content)
	case path == "/newcomment":
		postID, parseErr := id(r.PostForm.Get("postID"))
		content := strings.TrimSpace(r.PostForm.Get("content"))
		if parseErr != nil || !nonempty(content) || len(content) > 100000 {
			http.Error(w, "Invalid comment", 400)
			return
		}
		err = a.store.AddComment(user, postID, content)
		redirect = "/post/" + r.PostForm.Get("postID")
	default:
		parts := strings.Split(path, "/")
		if len(parts) != 3 {
			http.Error(w, "Invalid ID", 400)
			return
		}
		objectID, parseErr := id(parts[2])
		if parseErr != nil {
			http.Error(w, "Invalid ID", 400)
			return
		}
		comment := strings.HasSuffix(parts[1], "comm")
		// Use the actual parent post, never an unchecked redirect supplied by the caller.
		postID := objectID
		if comment {
			err = a.store.DB.QueryRow("SELECT postid FROM comments WHERE id=?", objectID).Scan(&postID)
		}
		if err == nil {
			err = a.store.Vote(user, comment, objectID, strings.HasPrefix(parts[1], "dislike"))
		}
		redirect = "/post/" + formatID(postID)
	}
	if err != nil {
		a.failure(w, err)
		return
	}
	http.Redirect(w, r, redirect, 303)
}

func formatID(id int64) string { return strconv.FormatInt(id, 10) }
