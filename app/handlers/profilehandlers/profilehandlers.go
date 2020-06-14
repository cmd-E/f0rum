package profilehandlers

import (
	"log"
	"net/http"
	"net/url"

	"github.com/IrvinIrvin/forum/app/cmanager"
	"github.com/IrvinIrvin/forum/app/contentmanager/postsmanager"
	"github.com/IrvinIrvin/forum/app/contentmanager/usermanager"
	"github.com/IrvinIrvin/forum/app/tools"
)

type profileStruct struct {
	User          usermanager.User
	RatedPostsN   int
	CreatedPostsN int
	UserID        int // отдельно так как хз как использовать 2 структуры вместе
}

//Profile handles profile page
func Profile(w http.ResponseWriter, r *http.Request) {
	log.Printf(" /// profile page")
	if _, err := cmanager.SessionExist(r); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	user := usermanager.GetUser(r)
	ratedPosts := postsmanager.RatedPostsNum(user.ID)
	createdPosts := postsmanager.CreatedPostsNum(user.ID)
	ProfileStruct := profileStruct{User: user, RatedPostsN: ratedPosts, CreatedPostsN: createdPosts, UserID: user.ID}
	err := tools.Templates.ExecuteTemplate(w, "profile.html", ProfileStruct)
	if err != nil {
		tools.ExecuteError(w, http.StatusInternalServerError, "Internal servaer error. Template error")
		log.Fatal(err.Error())
	}
	return
}

type createdPostsStruct struct {
	Posts []postsmanager.Post
	User  usermanager.User
}

// Createdpostsfunc shows created posts at page
func Createdpostsfunc(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(r.URL.Path)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	authorID := tools.GetID(u.Path)
	authorposts := postsmanager.GetPostsByAuthorID(authorID)
	authorpostsCutted := tools.CutContent(authorposts)
	user := usermanager.GetUser(r)
	createdPostsStr := createdPostsStruct{Posts: authorpostsCutted, User: user}
	err = tools.Templates.ExecuteTemplate(w, "createdposts.html", createdPostsStr)
	if err != nil {
		log.Fatal(" /// " + err.Error())
	}
}

type likedPostsStruct struct {
	Posts []postsmanager.Post
	User  usermanager.User
}

// RatedPostsFunc show liked posts at page
func RatedPostsFunc(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(r.URL.Path)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	userID := tools.GetID(u.Path)
	likedposts := postsmanager.GetRatedPosts(userID)
	likedpostsCutted := tools.CutContent(likedposts)
	user := usermanager.GetUser(r)
	likedPostsStr := likedPostsStruct{Posts: likedpostsCutted, User: user}
	err = tools.Templates.ExecuteTemplate(w, "createdposts.html", likedPostsStr)
	if err != nil {
		log.Fatal(" /// " + err.Error())
	}
}
