package reghandlers

import (
	"log"
	"net/http"

	"github.com/IrvinIrvin/forum/app/tools"
	"github.com/IrvinIrvin/forum/app/useractions"
)

// Registration handles registration page
func Registration(w http.ResponseWriter, r *http.Request) {
	log.Printf(" /// registration page")
	if useractions.AlreadyLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	err := tools.Templates.ExecuteTemplate(w, "registration.html", nil)
	if err != nil {
		tools.ExecuteError(w, http.StatusInternalServerError, "Internal servaer error. Template error")
		log.Fatal(err.Error())
	}
	return
}

type messageReg struct {
	Wrong     bool
	UserExist bool
}

// RegisterUser asks RegisterUserFunc (registerUser.go) to register new user
func RegisterUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		tools.ExecuteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if useractions.AlreadyLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	switch regResponse := useractions.RegisterUserFunc(w, r); regResponse {
	case 1: // Passwords doesn't match
		mssg := messageReg{Wrong: true, UserExist: false}
		err := tools.Templates.ExecuteTemplate(w, "registration.html", mssg)
		if err != nil {
			tools.ExecuteError(w, http.StatusInternalServerError, "Internal servaer error. Template error")
			log.Fatal(err.Error())
		}
	case 2: // User already exist
		mssg := messageReg{Wrong: false, UserExist: true}
		err := tools.Templates.ExecuteTemplate(w, "registration.html", mssg)
		if err != nil {
			tools.ExecuteError(w, http.StatusInternalServerError, "Internal servaer error. Template error")
			log.Fatal(err.Error())
		}
	case 3: // Error inserting user in database
		tools.ExecuteError(w, 500, "Ошибка в бд")
	case 4: // user credential are invalid (only spaces ex.)
		mssg := messageReg{Wrong: true, UserExist: false}
		err := tools.Templates.ExecuteTemplate(w, "registration.html", mssg)
		if err != nil {
			tools.ExecuteError(w, http.StatusInternalServerError, "Internal servaer error. Template error")
			log.Fatal(err.Error())
		}
	}
	return
}
