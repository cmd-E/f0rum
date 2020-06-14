package threadsmanager

import (
	"log"
	"net/http"
	"strconv"

	"github.com/IrvinIrvin/forum/app/dbconnect"
)

const tag = "threadsmanager.go"

// Thread struct of a single thread
type Thread struct {
	ID                 int
	Title, Description string
	PostsCount         int
}

// AddNewThread creates new thread
func AddNewThread(r *http.Request) {
	threadTitle := r.FormValue("threadTitle")
	threadDescription := r.FormValue("threadDescription")
	db := dbconnect.DBConn
	_, errS := db.Exec("INSERT INTO threads (title, description) VALUES (?, ?)", threadTitle, threadDescription)
	if errS != nil {
		log.Printf("AddNewThread (threadsmanager.go) Exec error: %s", errS.Error())
		return
	}
	log.Printf(tag + " /// thread added successfully")

}

// DeleteThread deletes existing thread
func DeleteThread(threadID string) {

	db := dbconnect.DBConn
	result, errS := db.Exec("DELETE FROM threads WHERE id=?", threadID)
	if errS != nil {
		log.Printf(tag+" /// Prepare statement error: %s", errS.Error())
		return
	}
	ar, err := result.RowsAffected()
	if err != nil {
		log.Println("result.RowsAffected() error: ", err.Error())
	}
	log.Println("Rows affected: ", ar)
	log.Printf(tag+" /// thread with id %s was successfully deleted", threadID)
}

// UpdateThread updates info of existing thread
func UpdateThread() {

}

// GetThreads gives all threads
func GetThreads() []Thread {
	var threads []Thread
	var idDB int
	var titleDB, descriptionDB string
	var postsCount int
	db := dbconnect.DBConn
	rows, err := db.Query("SELECT * FROM threads") //  id, title, description
	if err != nil {
		log.Fatal(err.Error())
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&idDB, &titleDB, &descriptionDB)
		postsCount = getCount(strconv.Itoa(idDB))
		//postsCount = 0
		//log.Printf(tag+" /// idDB: %d, titleDB: %s, descrDB: %s", idDB, titleDB, descriptionDB)
		threads = append(threads, Thread{ID: idDB, Title: titleDB, Description: descriptionDB, PostsCount: postsCount})
	}
	return threads
}

func getCount(threadID string) int {
	var count int
	db := dbconnect.DBConn
	rows, err := db.Query("SELECT COUNT(id) FROM postsandcats WHERE threadid=?", threadID)
	if err != nil {
		log.Println("getCounnt (threadsmanager.go) db.Query error: ", err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&count)
	}
	return count
}
