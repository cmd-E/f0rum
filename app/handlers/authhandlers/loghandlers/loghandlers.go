package loghandlers

import (
	"log"
	"net/http"

	"github.com/IrvinIrvin/forum/app/cmanager"
	"github.com/IrvinIrvin/forum/app/tools"
	"github.com/IrvinIrvin/forum/app/useractions"
)

// Login handles login page
func Login(w http.ResponseWriter, r *http.Request) {
	log.Printf("Login function (loghandlers.go)")
	if useractions.AlreadyLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	err := tools.Templates.ExecuteTemplate(w, "login.html", nil)
	if err != nil {
		tools.ExecuteError(w, http.StatusInternalServerError, "Internal servaer error. Template error")
		log.Fatal(err.Error())
	}
	return
}

type messageLogin struct {
	Wrong bool
}

// LoginUser asks LoginUserFunc (loginUser.go) to login users
func LoginUser(w http.ResponseWriter, r *http.Request) {
	log.Printf(" /// loginUser func")
	if r.Method != http.MethodPost {
		tools.ExecuteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if useractions.AlreadyLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if !useractions.LoginUserFunc(w, r) {
		mssg := messageLogin{Wrong: true}
		err := tools.Templates.ExecuteTemplate(w, "login.html", mssg)
		if err != nil {
			tools.ExecuteError(w, http.StatusInternalServerError, "Internal servaer error. Template error")
			log.Fatal(err.Error())
		}
	}
	return
}

// Logout deletes session from db and local storage
func Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		cmanager.DeleteSession(w, cookie)
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
