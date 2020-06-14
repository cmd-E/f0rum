package commhandlers

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/IrvinIrvin/forum/app/cmanager"
	"github.com/IrvinIrvin/forum/app/contentmanager/commentsmanager"
	"github.com/IrvinIrvin/forum/app/tools"
)

// NewComment creates new comment
func NewComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		tools.ExecuteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	postID := r.FormValue("postID")
	username := r.FormValue("username")
	content := r.FormValue("content")
	if commentsmanager.IsEmpty(content) {
		log.Printf(" /// got your empty comment")
		http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
		return
	}
	content = strings.Trim(content, " ")
	commentsmanager.AddComment(postID, username, content)
	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}

// DeleteComment deletes comment
func DeleteComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		tools.ExecuteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	commentID := r.FormValue("commentID")
	commentsmanager.DeleteComment(commentID)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// LikeComm likes comment
func LikeComm(w http.ResponseWriter, r *http.Request) {
	postID := r.FormValue("postID")
	_, errC := cmanager.SessionExist(r)
	if errC != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	log.Printf(" /// likecomm func")
	if r.Method != http.MethodPost {
		tools.ExecuteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	u, err := url.Parse(r.URL.Path)
	if err != nil {
		log.Fatal("url error" + err.Error())
		return
	}
	commID := tools.GetID(u.Path)
	commentsmanager.LikeComment(r, commID)
	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}

// DislikeComm dislikes comment
func DislikeComm(w http.ResponseWriter, r *http.Request) {
	postID := r.FormValue("postID")
	_, errC := cmanager.SessionExist(r)
	if errC != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	log.Printf(" /// dislikecomm func")
	if r.Method != http.MethodPost {
		tools.ExecuteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	u, err := url.Parse(r.URL.Path)
	if err != nil {
		log.Fatal("url error" + err.Error())
		return
	}
	commID := tools.GetID(u.Path)
	commentsmanager.DislikeComment(r, commID)
	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}
