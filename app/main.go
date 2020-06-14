package main

import (
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/IrvinIrvin/forum/app/contentmanager/usermanager"
	"github.com/IrvinIrvin/forum/app/dbconnect"
	"github.com/IrvinIrvin/forum/app/entities/ientities"
	"github.com/IrvinIrvin/forum/app/handlers/authhandlers/loghandlers"
	"github.com/IrvinIrvin/forum/app/handlers/authhandlers/reghandlers"
	"github.com/IrvinIrvin/forum/app/handlers/commhandlers"
	"github.com/IrvinIrvin/forum/app/handlers/postshandlers"
	"github.com/IrvinIrvin/forum/app/handlers/profilehandlers"
	"github.com/IrvinIrvin/forum/app/handlers/threadhandlers"
	"github.com/IrvinIrvin/forum/app/tools"

	_ "github.com/mattn/go-sqlite3"
)

func init() {
	tools.Templates = template.Must(template.ParseGlob("../templates/*.html"))
	dbconnect.DBConn = dbconnect.ConnectdbFunc()
}

func main() {

	http.HandleFunc("/", index)      // handles main page
	http.HandleFunc("/admin", admin) // handles admin page

	http.HandleFunc("/registration", reghandlers.Registration)
	http.HandleFunc("/reg", reghandlers.RegisterUser)

	http.HandleFunc("/login", loghandlers.Login)
	http.HandleFunc("/log", loghandlers.LoginUser)
	http.HandleFunc("/logout", loghandlers.Logout)

	http.HandleFunc("/profile", profilehandlers.Profile)
	http.HandleFunc("/createdposts/", profilehandlers.Createdpostsfunc)
	http.HandleFunc("/ratedposts/", profilehandlers.RatedPostsFunc)

	http.HandleFunc("/thread/", threadhandlers.Thread)
	http.HandleFunc("/newthread", threadhandlers.NewThread)
	http.HandleFunc("/deletethread", threadhandlers.DeleteThread)

	http.HandleFunc("/addpost", postshandlers.AddPostPage)
	http.HandleFunc("/newpost", postshandlers.NewPostFunc)
	http.HandleFunc("/deletepost", postshandlers.DeletePost)
	http.HandleFunc("/post/", postshandlers.Post)
	http.HandleFunc("/likepost/", postshandlers.LikeHandler)
	http.HandleFunc("/dislikepost/", postshandlers.DislikeHandler)

	http.HandleFunc("/newcomment", commhandlers.NewComment)
	http.HandleFunc("/deletecomment", commhandlers.DeleteComment)
	http.HandleFunc("/likecomm/", commhandlers.LikeComm)       // like comment func
	http.HandleFunc("/dislikecomm/", commhandlers.DislikeComm) //dislike comment func

	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("../css")))) // handles css folder
	// log.Println("Listening 8080...")
	// log.Println(http.ListenAndServe(":8080", nil))
	port := os.Getenv("PORT") //heroku присваивает свой порт сменить на свой при тестировании, либо экспортировать переменную PORT
	log.Println("heroku port: " + port)
	log.Println("Listening " + port + "...")
	log.Println(http.ListenAndServe(":"+port, nil))
}

// handles main page
func index(w http.ResponseWriter, r *http.Request) {
	log.Printf("index func (main.go)")
	if r.URL.Path != "/" {
		log.Printf("index func (main.go) Missing route: %s", r.URL.Path)
		tools.ExecuteError(w, http.StatusNotFound, "Page not found")
		return
	}
	IndexStruct := ientities.SetIndexStruct(r)
	err := tools.Templates.ExecuteTemplate(w, "index.html", IndexStruct)
	if err != nil {
		log.Fatal("main.go tools.Templates.ExecuteTemplate error: " + err.Error())
		tools.ExecuteError(w, http.StatusInternalServerError, "Internal server error. Template error")
		return
	}
	return
}

func admin(w http.ResponseWriter, r *http.Request) {
	if user := usermanager.GetUser(r); user.Name != "Deus" {
		tools.ExecuteError(w, http.StatusForbidden, "Admin page forbidden for all account except admin")
		return
	}
	err := tools.Templates.ExecuteTemplate(w, "admin.html", nil)
	if err != nil {
		tools.ExecuteError(w, http.StatusInternalServerError, "Internal servaer error. Template error")
		log.Fatal(err.Error())
	}
	return
}
