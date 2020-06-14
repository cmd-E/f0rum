package usermanager

import (
	"log"
	"net/http"

	"github.com/IrvinIrvin/forum/app/cmanager"
	"github.com/IrvinIrvin/forum/app/dbconnect"
)

const tag = "user.go"

// User - entity of user
type User struct {
	ID    int
	Name  string
	Email string
}

var u User

//GetUser returns pointer to User
func GetUser(r *http.Request) User {
	var u User
	c, err := cmanager.SessionExist(r)
	if err != nil {
		log.Printf("GetUser (usermanager.go) /// " + err.Error())
		return u
	}
	//log.Printf("c is %v", c.Value)
	db := dbconnect.DBConn
	rows, err := db.Query("SELECT username FROM sessions WHERE session=?", c.Value)
	if err != nil {
		log.Printf(tag + " /// querry error found")
	}
	defer rows.Close()
	var ID int
	var username string
	var email string
	for rows.Next() {
		rows.Scan(&username)
	}
	rows, err = db.Query("SELECT id, email FROM users WHERE username=?", username)
	if err != nil {
		log.Printf(tag + " /// querry error found")
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&ID, &email)
	}
	rows.Close()
	//log.Printf("ID:" + strconv.Itoa(ID) + ", Name:" + username + ", Email:" + email)
	u = User{ID: ID, Name: username, Email: email}
	//log.Printf("u is %v", u)
	return u
}
