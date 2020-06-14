package useractions

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/IrvinIrvin/forum/app/cmanager"
	"github.com/IrvinIrvin/forum/app/dbconnect"

	"golang.org/x/crypto/bcrypt"
)

// LoginUserFunc (loginUser.go) logins users
func LoginUserFunc(w http.ResponseWriter, r *http.Request) bool {
	log.Println(tag + " /// LoginUserFunc")
	username := r.FormValue("username")
	password := r.FormValue("password")
	//database := dbconnect.ConnectdbFunc()
	database := dbconnect.DBConn
	passDB, errExistance := isUserExist(database, username)
	if errExistance != nil {
		log.Printf("Неверное имя пользователя или пароль")
		return false
	}
	if !comparePasswords(passDB, []byte(password)) {
		log.Printf("Неверное имя пользователя или пароль")
		return false
	}
	existedSessions, areExist := cmanager.IsSessionForUserExist(database, username)
	if areExist {
		cmanager.DeleteOldSessions(database, existedSessions)
	}
	cmanager.SetSession(username, database, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return true
}

func getUserEmail(db *sql.DB, username string) string {
	rows, err := db.Query("SELECT email FROM users WHERE username=?", username)
	if err != nil {
		log.Printf("Query error: %s", err.Error())
		return ""
	}
	defer rows.Close()
	var emailDB string
	for rows.Next() {
		rows.Scan(&emailDB)
	}
	return emailDB
}

// AlreadyLoggedIn checks if user already logged in
func AlreadyLoggedIn(r *http.Request) bool {
	log.Println("AlreadyLoggedIn function (loginUser.go)")
	_, errCookie := cmanager.SessionExist(r)
	if errCookie != nil {
		log.Printf("AlreadyLoggedIn (loginUser.go) cmanager.SessionExist(r) error: " + errCookie.Error())
		return false
	}
	log.Println("AlreadyLoggedIn (loginUser.go): User already has session in db and local storage")
	return true
}

// isUserExist check if user exists
func isUserExist(database *sql.DB, username string) (string, error) {
	rows, err := database.Query("SELECT username, password FROM users")
	if err != nil {
		log.Printf("isUserExist (loginUser.go) database.Query error: %s", err.Error())
	}
	defer rows.Close()
	var usernameDB string
	var passwordDB string
	for rows.Next() {
		rows.Scan(&usernameDB, &passwordDB)
		//log.Println(usernameDB + " " + passwordDB)
		if username == usernameDB {
			return passwordDB, nil
		}
	}
	return "", errors.New("user not found")
}

func comparePasswords(hashedPwd string, plainPwd []byte) bool {
	byteHash := []byte(hashedPwd)
	err := bcrypt.CompareHashAndPassword(byteHash, plainPwd)
	if err != nil {
		log.Println(err)
		return false
	}

	return true
}
