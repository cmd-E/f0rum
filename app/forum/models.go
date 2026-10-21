package forum

import (
	"database/sql"
	"errors"
	"strings"
)

var ErrForbidden = errors.New("forbidden")
var ErrConflict = errors.New("already exists")

type User struct {
	ID                int64
	Name, Email, Role string
}
type Thread struct {
	ID                 int64
	Title, Description string
	PostsCount         int
}
type Post struct {
	ID              int64
	Title, Content  string
	Likes, Dislikes int
	Author          string
	AuthorID        int64
}
type Comment struct {
	ID, PostID        int64
	Username, Content string
	Likes, Dislikes   int
}
type View struct {
	User                         User
	Name, Email, CSRF            string
	UserID                       int64
	Threads                      []Thread
	Posts                        []Post
	Post                         interface{}
	Comments                     []Comment
	RatedPostsN, CreatedPostsN   int
	Wrong, UserExist             bool
	Message, Errcode, Errmessage string
}

func Preview(content string) string {
	r := []rune(content)
	if len(r) > 90 {
		return string(r[:90]) + "..."
	}
	return content
}
func nonempty(value string) bool { return strings.TrimSpace(value) != "" }
func affected(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
