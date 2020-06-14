package postsmanager

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/IrvinIrvin/forum/app/contentmanager/usermanager"
	"github.com/IrvinIrvin/forum/app/dbconnect"
)

const tag = "postsmanager.go"

// Post - struct of a single post
type Post struct {
	ID              int
	Title, Content  string
	Likes, Dislikes int
	Author          string
	AuthorID        int
}

// AddPost creates new post and puts it into db
func AddPost(threadsID []string, postTitle, postContent, author, authorID string) {
	database := dbconnect.DBConn
	result, err := database.Exec("INSERT INTO posts (title, content, author, authorid) VALUES (?, ?, ?, ?)", postTitle, postContent, author, authorID)
	if err != nil {
		log.Printf("Add post (postsmanager.go) db.Exec error: %s", err.Error())
		return
	}
	lastInsertedID, err := result.LastInsertId()
	if err != nil {
		log.Println("result.LastInsertId() error (postsmanager.go): ", err.Error())
	}
	addCatsToPost(threadsID, strconv.FormatInt(lastInsertedID, 10))
	log.Printf(tag + " /// post added successfully")
}

func addCatsToPost(threadsID []string, id string) {
	database := dbconnect.DBConn
	for _, thread := range threadsID {
		_, err := database.Exec("INSERT INTO postsandcats (threadid, postid) VALUES (?, ?)", thread, id)
		if err != nil {
			log.Printf("addCatsToPost db.Exec error: %s", err.Error())
			return
		}
	}
	log.Printf("All cats was inserted in db for id %s", id)

}

// DeletePost deletes Post with postID
func DeletePost(postID string) {
	db := dbconnect.DBConn
	statement, errS := db.Prepare("DELETE FROM posts WHERE id=" + postID)
	if errS != nil {
		log.Printf("Prepare statement error: %s", errS.Error())
		return
	}
	defer statement.Close()
	statement.Exec()
	log.Printf(tag+" /// post with id %s was deleted", postID)
}

// GetPostsByThreadID gets post of certain thread
func GetPostsByThreadID(ThreadID string) []Post {
	var posts []Post
	var postIDDB int
	db := dbconnect.DBConn
	//rows, err := db.Query("SELECT id, threadid, title, content, likes, dislikes, author FROM posts WHERE threadid=" + ThreadID)
	rows, err := db.Query("SELECT postid FROM postsandcats WHERE threadid=?", ThreadID)
	if err != nil {
		log.Fatal("GetPostsByThreadID (postsmanager.go) db.Query error: ", err.Error())
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&postIDDB)
		posts = append(posts, GetPostByItsID(strconv.Itoa(postIDDB)))
	}
	return posts
}

// GetPostByItsID gets post by it's id
func GetPostByItsID(postID string) Post {
	var post Post
	var idDB int
	var titleDB, contentDB string
	var likesDB, dislikesDB int
	var authorDB string
	var authorIDDB int
	db := dbconnect.DBConn
	//rows, err := db.Query("SELECT id, threadid, title, content, likes, dislikes, author FROM posts WHERE id=" + postID)
	// rows, err := db.Query("SELECT * FROM posts WHERE id=?", postID)
	// if err != nil {
	// 	log.Fatal(err.Error())
	// 	return Post{}
	// }
	// defer rows.Close()
	// for rows.Next() {
	// 	rows.Scan(&idDB, &titleDB, &contentDB, &likesDB, &dislikesDB, &authorDB, &authorIDDB)

	// 	//log.Printf(tag+" /// idDB: %d, threadIDDB: %d, titleDB: %s, contentDB: %s, likesDB: %d, dislikesDB: %d", idDB, threadIDDB, titleDB, contentDB, likesDB, dislikesDB)
	// }
	row := db.QueryRow("SELECT id, title, content, likes, dislikes, author, authorid FROM posts WHERE id=?", postID)
	err := row.Scan(&idDB, &titleDB, &contentDB, &likesDB, &dislikesDB, &authorDB, &authorIDDB)
	if err != nil {
		log.Println("GetPostByItsID row scan error: ", err.Error())
		return post
	}
	// log.Printf("authorDB: %v", authorDB)
	post = Post{ID: idDB, Title: titleDB, Content: contentDB, Likes: likesDB, Dislikes: dislikesDB, Author: authorDB, AuthorID: authorIDDB}
	return post
}

// GetPostsByAuthorID returns posts by author id
func GetPostsByAuthorID(authorID string) []Post {
	var posts []Post
	var idDB int
	var titleDB, contentDB string
	var likesDB, dislikesDB int
	var authorDB string
	var authorIDDB int
	db := dbconnect.DBConn
	rows, err := db.Query("SELECT * FROM posts WHERE authorid=?", authorID)
	if err != nil {
		log.Fatal(tag + " /// " + err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&idDB, &titleDB, &contentDB, &likesDB, &dislikesDB, &authorDB, &authorIDDB)
		posts = append(posts, Post{ID: idDB, Title: titleDB, Content: contentDB, Likes: likesDB, Dislikes: dislikesDB, Author: authorDB, AuthorID: authorIDDB})
	}
	return posts
}

// GetRatedPosts gets all lkied posts
func GetRatedPosts(userID string) []Post {
	var posts []Post
	var postID string
	db := dbconnect.DBConn
	rows, err := db.Query("SELECT postid FROM rtactions WHERE userid=?", userID)
	if err != nil {
		log.Fatal(tag + " /// " + err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&postID)
		posts = append(posts, GetPostByItsID(postID))
	}
	return posts
}

// LikePost increment likes by 1
func LikePost(r *http.Request, postID string) {
	db := dbconnect.DBConn
	user := usermanager.GetUser(r)
	if isRated(db, user.Name, postID) {
		log.Printf(tag + " /// already rated")
		return
	}
	_, err := db.Exec("UPDATE posts SET likes=likes+1 WHERE id=?", postID)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	addToRtactions(db, user, postID, "like")
	log.Printf("Successfully liked")
}

// DislikePost increment dislikes by 1
func DislikePost(r *http.Request, postID string) {
	db := dbconnect.DBConn
	user := usermanager.GetUser(r)
	if isRated(db, user.Name, postID) {
		log.Printf(tag + " /// already rated")
		return
	}
	_, err := db.Exec("UPDATE posts SET dislikes=dislikes+1 WHERE id=?", postID)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	addToRtactions(db, user, postID, "dislike")
	log.Printf("Successfully disliked")
}

func isRated(db *sql.DB, username, postID string) bool {
	rows, err := db.Query("SELECT * FROM rtactions WHERE postid=? AND username=?", postID, username)
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

func addToRtactions(db *sql.DB, user usermanager.User, postID, rateType string) {
	log.Printf(tag + " /// addToRtactions func")
	statement, errS := db.Prepare("INSERT INTO rtactions (postid, username, userid, type) VALUES (?, ?, ?, ?)")
	if errS != nil {
		log.Printf("Prepare statement error: %s", errS.Error())
		return
	}
	defer statement.Close()
	statement.Exec(postID, user.Name, user.ID, rateType)
	log.Printf("action added to rateactions")
}

// CreatedPostsNum returns number of created posts
func CreatedPostsNum(authorID int) int {
	log.Println("CreatedPostsNum")
	postNum := 0
	createdPosts := GetPostsByAuthorID(strconv.Itoa(authorID))
	for range createdPosts {
		postNum++
	}
	log.Println("postNum = " + strconv.Itoa(postNum))
	return postNum
}

// RatedPostsNum returns number of rated posts
func RatedPostsNum(authorID int) int {
	log.Println("RatedPostsNum")
	postNum := 0
	ratedPosts := GetRatedPosts(strconv.Itoa(authorID))
	for _, post := range ratedPosts {
		log.Println(post.Author)
		postNum++
	}
	log.Println("rated postNum = " + strconv.Itoa(postNum))
	return postNum
}
