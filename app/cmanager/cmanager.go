package cmanager

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/IrvinIrvin/forum/app/dbconnect"
	uuid "github.com/satori/go.uuid"
)

const tag = "cmanager.go"

// SessionExist checks if session's cookie exists
func SessionExist(r *http.Request) (*http.Cookie, error) {
	c, err := r.Cookie("session")
	if err != nil {
		return nil, errors.New("Cookie not found in local storage")
	}
	var session string
	var sessionNum int
	// db := dbconnect.ConnectdbFunc()
	rows, err := dbconnect.DBConn.Query("SELECT session FROM sessions WHERE session=?", c.Value)
	if err != nil {
		log.Fatalf("SessionExist (cmanager.go) error: %s", err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&session)
		if session != "" {
			sessionNum++
			if session == c.Value {
				log.Println("SessionExist (cmanager.go): found session that match cookie")
				return c, nil
			}
			log.Println("SessionExist (cmanager.go) Got non empty session: " + session)
		}
		session = ""
	}
	if sessionNum >= 0 {
		return nil, errors.New("Cookie not found in database")
	}
	return c, nil
}

// SetSession sets new session and returns cookie
func SetSession(username string, database *sql.DB, w http.ResponseWriter) {
	sessionID := uuid.NewV4()
	c := &http.Cookie{
		Name:  "session",
		Value: sessionID.String(),
	}
	http.SetCookie(w, c)
	log.Printf("SetSession (cmanager.go) Cookie %v is set locally", c.Value)
	statement, errS := database.Prepare("INSERT INTO sessions (session, username) VALUES (?, ?)")
	if errS != nil {
		log.Printf("SetSession (cmanager.go) Prepare statement error: %s", errS.Error())
		return
	}
	defer statement.Close()
	statement.Exec(c.Value, username)
	log.Printf("SetSession (cmanager.go) Session inserted in database")
}

// DeleteSession deletes cookie of session
func DeleteSession(w http.ResponseWriter, sessionCookie *http.Cookie) {
	cookie := &http.Cookie{
		Name:   "session",
		Value:  "0",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
	DeleteSessionDB(dbconnect.DBConn, sessionCookie.Value)
	log.Println("DeleteSession (cmanager.go) session was deleted")
}

// IsSessionForUserExist checks if session for user is exist
func IsSessionForUserExist(db *sql.DB, username string) ([]string, bool) {
	var existingSessions []string
	var session string
	var sessionNum int
	rows, err := db.Query("SELECT session FROM sessions WHERE username=?", username)
	if err != nil {
		log.Fatalf("SessionForUserExist (cmanager.go) error: " + err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&session)
		if session != "" {
			sessionNum++
			existingSessions = append(existingSessions, session)
			//log.Println("session num is " + strconv.Itoa(sessionNum))
			log.Println("IsSessionForUserExist (cmanager.go) Got non empty session: " + session)
		}
		session = ""
	}
	if sessionNum > 0 {
		return existingSessions, true
	}
	return nil, false
}

// DeleteOldSessions initializes DeleteSessionDB which deletes list of sessions from database
func DeleteOldSessions(database *sql.DB, oldSessions []string) {
	for _, oldSession := range oldSessions {
		DeleteSessionDB(database, oldSession)
	}
}

// DeleteSessionDB deletes sessions from database
func DeleteSessionDB(db *sql.DB, session string) {
	result, errS := db.Exec("DELETE FROM sessions WHERE session=?", session)
	if errS != nil {
		log.Printf("DeleteSessionDB db.Exec (cmanager) error: %v", errS.Error())
		return
	}
	log.Printf(tag + " /// session successfuly removed")
	ra, _ := result.RowsAffected()
	log.Println("Rows affected: ", ra)
}
