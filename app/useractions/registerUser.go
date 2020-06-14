package useractions

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/IrvinIrvin/forum/app/tools"

	"github.com/IrvinIrvin/forum/app/cmanager"
	dbconnect "github.com/IrvinIrvin/forum/app/dbconnect"

	"golang.org/x/crypto/bcrypt"
)

const tag = "useractions"

// RegisterUserFunc (registerUser.go) registers new users
func RegisterUserFunc(w http.ResponseWriter, r *http.Request) int { // коды ошибок: 1 - поля неверно заполнены, 2 - пользователь существует, 3 - ошибка сервера
	log.Printf(tag + " /// registerUser func")
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	passwordRepeat := r.FormValue("password_repeat")
	if !tools.ValidateUnameAndPass(username, password) {
		log.Println("Invalide username or password")
		return 4
	}
	if password != passwordRepeat {
		log.Printf(tag + " /// Passwords don't match")
		//http.Redirect(w, r, "/registration", http.StatusSeeOther)
		return 1
	}
	database := dbconnect.DBConn
	if isRegistered(database, username, email) {
		log.Printf(tag + " /// User already registered")
		//http.Redirect(w, r, "/registration", http.StatusSeeOther)
		return 2
	}
	passwordEnc := HashAndSalt([]byte(password))
	log.Printf(tag+"///"+`	username: %s
	password: %s
	repeat password: %s
	passwordEnc: %s
	email: %s`,
		username,
		password,
		passwordRepeat,
		passwordEnc,
		email)

	_, err := database.Exec("INSERT INTO users (username, email, password) VALUES (?, ?, ?)", username, email, passwordEnc)
	if err != nil {
		log.Fatalf("Prepare statement error: %s", err.Error())
		return 3
	}
	log.Printf(tag+" /// db updated with %s, %s, %s", username, passwordEnc, email)
	cmanager.SetSession(username, database, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return 0
}

// isRegistered checks if username or email in the DB
func isRegistered(database *sql.DB, username string, email string) bool {
	row := database.QueryRow("SELECT id FROM users WHERE username=? OR email=?", username, email)
	var usernameDB string
	var emailDB string
	err := row.Scan(&usernameDB, &emailDB)
	if err == sql.ErrNoRows {
		return false
	}
	if username == usernameDB || email == emailDB {
		return true
	}
	return false
}

// HashAndSalt encrypts passwords
func HashAndSalt(pwd []byte) string {
	hash, err := bcrypt.GenerateFromPassword(pwd, bcrypt.MinCost)
	if err != nil {
		log.Println(err)
	}
	return string(hash)
}
