package postshandlers

import (
	"log"
	"net/http"
	"net/url"

	"github.com/IrvinIrvin/forum/app/cmanager"
	"github.com/IrvinIrvin/forum/app/contentmanager/commentsmanager"
	"github.com/IrvinIrvin/forum/app/contentmanager/postsmanager"
	"github.com/IrvinIrvin/forum/app/contentmanager/threadsmanager"
	"github.com/IrvinIrvin/forum/app/contentmanager/usermanager"
	"github.com/IrvinIrvin/forum/app/tools"
)

type commentsSectionStruct struct {
	Name, Email string
	Post        postsmanager.Post
	Comments    []commentsmanager.Comment
	Message     string
}

// Post handles post page
func Post(w http.ResponseWriter, r *http.Request) {
	log.Printf(" /// post function")
	u, err := url.Parse(r.URL.Path)
	if err != nil {
		log.Fatal("url error" + err.Error())
		return
	}
	postID := tools.GetID(u.Path)
	if postID == "" {
		tools.ExecuteError(w, http.StatusForbidden, "Forbidden")
		return
	}
	user := usermanager.GetUser(r)
	post := postsmanager.GetPostByItsID(postID)
	comments := commentsmanager.GetCommentsByPostID(postID)
	CommentsSectionStruct := commentsSectionStruct{Name: user.Name, Email: user.Email, Post: post, Comments: comments}
	errT := tools.Templates.ExecuteTemplate(w, "post.html", CommentsSectionStruct)
	if errT != nil {
		log.Fatal(" /// template error: " + errT.Error())
	}
	return
}

// LikeHandler likes post
func LikeHandler(w http.ResponseWriter, r *http.Request) {
	_, errC := cmanager.SessionExist(r)
	if errC != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	log.Printf("LikeHandler (postshandler.go)")
	if r.Method != http.MethodPost {
		tools.ExecuteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	u, err := url.Parse(r.URL.Path)
	if err != nil {
		log.Fatal("url error" + err.Error())
		return
	}
	postID := tools.GetID(u.Path)
	postsmanager.LikePost(r, postID)
	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
	return
}

//DislikeHandler dislike post
func DislikeHandler(w http.ResponseWriter, r *http.Request) {
	_, errC := cmanager.SessionExist(r)
	if errC != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	log.Printf(" /// dislikehandler func")
	if r.Method != http.MethodPost {
		tools.ExecuteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	u, err := url.Parse(r.URL.Path)
	if err != nil {
		log.Fatal("url error" + err.Error())
		return
	}
	postID := tools.GetID(u.Path)
	postsmanager.DislikePost(r, postID)
	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}

type addpoststruct struct {
	User    usermanager.User
	Threads []threadsmanager.Thread
}

// AddPostPage handles post creation page
func AddPostPage(w http.ResponseWriter, r *http.Request) {
	if _, err := cmanager.SessionExist(r); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	threads := threadsmanager.GetThreads()
	user := usermanager.GetUser(r)
	postCreationStruct := addpoststruct{User: user, Threads: threads}
	err := tools.Templates.ExecuteTemplate(w, "addpost.html", postCreationStruct)
	if err != nil {
		log.Fatal(" /// " + err.Error())
	}
}

// NewPostFunc function to create post
func NewPostFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		tools.ExecuteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	authorID := r.FormValue("authorID")
	author := r.FormValue("postAuthor")
	threadsID := r.Form["category"] // returns slice of threadsID
	if count := tools.ThreadsCount(threadsID); count < 1 {
		// TODO Send message to user
		http.Redirect(w, r, "/addpost", http.StatusSeeOther)
		return
	}
	postTitle := r.FormValue("postTitle")
	postContent := r.FormValue("postContent")
	if tools.IsEmpty(postTitle) || tools.IsEmpty(postContent) {
		// TODO Send message to user
		http.Redirect(w, r, "/addpost", http.StatusSeeOther)
		return
	}
	postsmanager.AddPost(threadsID, postTitle, postContent, author, authorID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	log.Printf("NewPostFunc (postshandler.go) author is %s, it's id is %s, threads are %v, postTitle is %s, content is %s", author, authorID, threadsID, postTitle, postContent)
}

// DeletePost deletes post
func DeletePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		tools.ExecuteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	postID := r.FormValue("postID")
	postsmanager.DeletePost(postID)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
