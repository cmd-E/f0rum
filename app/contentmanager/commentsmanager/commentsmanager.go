package commentsmanager

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/IrvinIrvin/forum/app/contentmanager/usermanager"
	"github.com/IrvinIrvin/forum/app/dbconnect"
)

// Comment struct of a single comment
type Comment struct {
	ID, PostID        int
	Username, Content string
	Likes, Dislikes   int
}

const tag = "commentsmanager.go"

// AddComment adds new comment
func AddComment(postID, username, content string) {
	db := dbconnect.DBConn
	_, errS := db.Exec("INSERT INTO comments (postid, username, content) VALUES (?, ?, ?)", postID, username, content)
	if errS != nil {
		log.Printf("Prepare statement error: %s", errS.Error())
		return
	}
	log.Printf(tag + " /// comment added successfully")
}

// IsEmpty check if comment is only spaces
func IsEmpty(comment string) bool {
	count := 0
	for _, l := range comment {
		if l != ' ' {
			count++
		}
	}
	return count == 0
}

// DeleteComment deletes comment
func DeleteComment(commentID string) {
	db := dbconnect.DBConn
	_, errS := db.Exec("DELETE FROM comments WHERE id=?", commentID)
	if errS != nil {
		log.Printf(tag+" /// Prepare statement error: %s", errS.Error())
		return
	}
	log.Printf(tag+" /// comment with id %s was deleted successfuly", commentID)
}

// GetCommentsByPostID returnes slice of comments by post id
func GetCommentsByPostID(PostID string) []Comment {
	var comments []Comment
	var idDB, postIDDB int
	var usernameDB, contentDB string
	var likesDB, dislikesDB int
	db := dbconnect.DBConn
	rows, err := db.Query("SELECT * FROM comments WHERE postid=?", PostID)
	if err != nil {
		log.Fatal(err.Error())
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&idDB, &postIDDB, &usernameDB, &contentDB, &likesDB, &dislikesDB)
		//log.Printf(tag+" /// idDB: %d, postIDDB: %d, usernameDB: %s, contentDB: %s, likesDB: %d, dislikesDB: %d", idDB, postIDDB, usernameDB, contentDB, likesDB, dislikesDB)
		comments = append(comments, Comment{ID: idDB, PostID: postIDDB, Username: usernameDB, Content: contentDB, Likes: likesDB, Dislikes: dislikesDB})
	}
	return comments
}

// LikeComment adds like to comment
func LikeComment(r *http.Request, commID string) {
	db := dbconnect.DBConn
	user := usermanager.GetUser(r)
	if isCommRated(db, user.Name, commID) {
		log.Printf(tag + " /// already rated")
		return
	}
	_, err := db.Exec("UPDATE comments SET likes=likes+1 WHERE id=" + commID)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	addToRtactionsComm(db, user.Name, commID)
	log.Printf(tag + " /// Comment successfully liked")
}

// DislikeComment removes like from comment
func DislikeComment(r *http.Request, commID string) {
	db := dbconnect.DBConn
	user := usermanager.GetUser(r)
	if isCommRated(db, user.Name, commID) {
		log.Printf(tag + " /// already rated")
		return
	}
	_, err := db.Exec("UPDATE comments SET dislikes=dislikes+1 WHERE id=" + commID)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	addToRtactionsComm(db, user.Name, commID)
	log.Printf(tag + " /// Comment successfully liked")
}

func isCommRated(db *sql.DB, username, commID string) bool {
	rows, err := db.Query("SELECT * FROM rtactionscomm WHERE commid=? AND username=?", commID, username)
	if err != nil {
		log.Fatal(err.Error())
		return true
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
	}
	return count > 0
}

func addToRtactionsComm(db *sql.DB, username, commID string) {
	log.Printf(tag + " /// addToRtactionsComm func")
	statement, errS := db.Prepare("INSERT INTO rtactionscomm (commid, username) VALUES (?, ?)")
	if errS != nil {
		log.Printf("Prepare statement error: %s", errS.Error())
		return
	}
	defer statement.Close()
	statement.Exec(commID, username)
	log.Printf("action added to rateactionscomm")
}
