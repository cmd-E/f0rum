package threadhandlers

import (
	"log"
	"net/http"
	"net/url"

	"github.com/IrvinIrvin/forum/app/contentmanager/postsmanager"
	"github.com/IrvinIrvin/forum/app/contentmanager/threadsmanager"
	"github.com/IrvinIrvin/forum/app/contentmanager/usermanager"
	"github.com/IrvinIrvin/forum/app/tools"
)

type postsStruct struct {
	Name string
	Post []postsmanager.Post
}

// Thread handles "localhost:8080/thread/*" page
func Thread(w http.ResponseWriter, r *http.Request) {
	log.Printf("Thread (thandlers.go)")
	u, err := url.Parse(r.URL.Path)
	if err != nil {
		log.Fatal(err.Error())
	}
	threadID := tools.GetID(u.Path)
	if threadID == "" {
		log.Printf("Thread (thandlers.go): threadID is empty")
		tools.ExecuteError(w, http.StatusForbidden, "Forbidden")
		return
	}
	user := usermanager.GetUser(r)
	posts := postsmanager.GetPostsByThreadID(threadID)
	cuttedposts := tools.CutContent(posts)
	PostsStruct := postsStruct{Name: user.Name, Post: cuttedposts}
	errT := tools.Templates.ExecuteTemplate(w, "thread.html", PostsStruct)
	if errT != nil {
		log.Fatal(errT.Error())
	}
	return
}

// NewThread creates new thread (works from admin panel yet)
func NewThread(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		tools.ExecuteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	threadsmanager.AddNewThread(r)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// DeleteThread deletes thread (works from admin panel yet)
func DeleteThread(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		tools.ExecuteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	threadID := r.FormValue("threadID")
	threadsmanager.DeleteThread(threadID)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
