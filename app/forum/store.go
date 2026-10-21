package forum

import (
	"database/sql"
	_ "embed"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed schema.sql
var schema string

type Store struct{ DB *sql.DB }

func Open(path string) (*Store, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	uri := databaseURI(absolute)
	db, err := sql.Open("sqlite3", uri.String()+"?_foreign_keys=on&_busy_timeout=5000&_txlock=immediate")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("open SQLite (CGO and a C compiler are required): %w", err)
	}
	return &Store{db}, nil
}

func databaseURI(absolute string) url.URL {
	path := filepath.ToSlash(absolute)
	// A drive-letter path needs a leading slash to remain a URI path,
	// rather than accidentally becoming the authority in file://C:/... .
	if filepath.VolumeName(absolute) != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return url.URL{Scheme: "file", Path: path}
}
func (s *Store) Version() (int, error) {
	var version int
	err := s.DB.QueryRow("PRAGMA user_version").Scan(&version)
	return version, err
}
func (s *Store) Initialize() error {
	version, err := s.Version()
	if err != nil {
		return err
	}
	if version == 1 {
		return nil
	}
	if version != 0 {
		return fmt.Errorf("unsupported schema version %d", version)
	}
	var count int
	if err = s.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return fmt.Errorf("legacy database: back up and run with -migrate; conflicting records must be reconciled explicitly")
	}
	return s.migrate(false, 0)
}

// Migrate creates a SQLite-consistent backup before changing an existing database.
// It never chooses a duplicate account, promotes a name, or silently drops content.
func (s *Store) Migrate(path string, adminID int64) error {
	version, err := s.Version()
	if err != nil {
		return err
	}
	if version > 1 {
		return fmt.Errorf("unsupported schema version %d", version)
	}
	if version == 1 {
		if adminID != 0 {
			return affected(s.DB.Exec("UPDATE users SET role='admin' WHERE id=?", adminID))
		}
		return nil
	}
	backup := path + fmt.Sprintf(".backup-%d", time.Now().UnixNano())
	if _, err = os.Stat(backup); !os.IsNotExist(err) {
		return fmt.Errorf("backup destination is unavailable")
	}
	if _, err = s.DB.Exec("VACUUM INTO ?", backup); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	var count int
	if err = s.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count); err != nil {
		return err
	}
	return s.migrate(count != 0, adminID)
}
func (s *Store) migrate(legacy bool, adminID int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(schema); err != nil {
		return err
	}
	if legacy {
		statements := []string{
			"INSERT INTO users_new(id,username,email,password) SELECT id,username,email,password FROM users",
			"INSERT INTO threads_new SELECT id,title,description FROM threads",
			"INSERT INTO posts_new SELECT id,title,content,COALESCE(likes,0),COALESCE(dislikes,0),authorid FROM posts",
			"INSERT INTO postsandcats_new SELECT id,threadid,postid FROM postsandcats",
			"INSERT INTO comments_new SELECT id,postid,(SELECT id FROM users_new WHERE username=comments.username),content,COALESCE(likes,0),COALESCE(dislikes,0) FROM comments",
			"INSERT INTO rtactions_new SELECT id,postid,(SELECT id FROM users_new WHERE username=rtactions.username),type FROM rtactions",
			// Legacy comment votes have no direction. NULL preserves that uncertainty.
			"INSERT INTO rtactionscomm_new(id,commid,user_id) SELECT id,commid,(SELECT id FROM users_new WHERE username=rtactionscomm.username) FROM rtactionscomm",
		}
		for _, statement := range statements {
			if _, err = tx.Exec(statement); err != nil {
				return fmt.Errorf("migration stopped; duplicate/orphan/invalid records require reconciliation: %w", err)
			}
		}
		for _, table := range []string{"rtactionscomm", "rtactions", "comments", "postsandcats", "posts", "threads", "sessions", "users"} {
			if _, err = tx.Exec("DROP TABLE " + table); err != nil {
				return err
			}
		}
	}
	for _, table := range []string{"users", "sessions", "threads", "posts", "postsandcats", "comments", "rtactions", "rtactionscomm"} {
		if _, err = tx.Exec("ALTER TABLE " + table + "_new RENAME TO " + table); err != nil {
			return err
		}
	}
	if adminID != 0 {
		if err = affected(tx.Exec("UPDATE users SET role='admin' WHERE id=?", adminID)); err != nil {
			return fmt.Errorf("administrator ID not found: %w", err)
		}
	}
	if _, err = tx.Exec("CREATE INDEX sessions_expiry ON sessions(expires_at); CREATE INDEX sessions_user ON sessions(user_id); CREATE INDEX posts_author ON posts(authorid); CREATE INDEX comments_post ON comments(postid); CREATE INDEX postsandcats_post ON postsandcats(postid); CREATE INDEX rtactions_post ON rtactions(postid); CREATE INDEX rtactionscomm_comment ON rtactionscomm(commid); PRAGMA user_version=1"); err != nil {
		return err
	}
	rows, err := tx.Query("PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	invalid := rows.Next()
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if invalid {
		return fmt.Errorf("foreign key violations; migration rolled back")
	}
	return tx.Commit()
}
func (s *Store) User(id int64) (User, error) {
	var u User
	err := s.DB.QueryRow("SELECT id,username,email,role FROM users WHERE id=?", id).Scan(&u.ID, &u.Name, &u.Email, &u.Role)
	return u, err
}
func (s *Store) Threads() ([]Thread, error) {
	rows, err := s.DB.Query("SELECT t.id,t.title,t.description,COUNT(p.id) FROM threads t LEFT JOIN postsandcats p ON p.threadid=t.id GROUP BY t.id,t.title,t.description ORDER BY t.id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Thread{}
	for rows.Next() {
		var t Thread
		if err = rows.Scan(&t.ID, &t.Title, &t.Description, &t.PostsCount); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

const postColumns = "SELECT p.id,p.title,p.content,p.likes,p.dislikes,u.username,p.authorid FROM posts p JOIN users u ON u.id=p.authorid "

func (s *Store) Posts(filter string, id int64) ([]Post, error) {
	// Query fragments are chosen internally, never supplied by HTTP input.
	query := postColumns
	args := []interface{}{}
	switch filter {
	case "thread":
		query += "JOIN postsandcats c ON c.postid=p.id WHERE c.threadid=? "
		args = append(args, id)
	case "author":
		query += "WHERE p.authorid=? "
		args = append(args, id)
	case "rated":
		query += "JOIN rtactions r ON r.postid=p.id WHERE r.user_id=? "
		args = append(args, id)
	case "id":
		query += "WHERE p.id=? "
		args = append(args, id)
	default:
		return nil, fmt.Errorf("unknown post query")
	}
	rows, err := s.DB.Query(query+"ORDER BY p.id", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Post{}
	for rows.Next() {
		var p Post
		if err = rows.Scan(&p.ID, &p.Title, &p.Content, &p.Likes, &p.Dislikes, &p.Author, &p.AuthorID); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
func (s *Store) Comments(id int64) ([]Comment, error) {
	rows, err := s.DB.Query("SELECT c.id,c.postid,u.username,c.content,c.likes,c.dislikes FROM comments c JOIN users u ON u.id=c.user_id WHERE c.postid=? ORDER BY c.id", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Comment{}
	for rows.Next() {
		var c Comment
		if err = rows.Scan(&c.ID, &c.PostID, &c.Username, &c.Content, &c.Likes, &c.Dislikes); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}
func (s *Store) AddPost(user User, categories []int64, title, content string) error {
	if user.ID == 0 {
		return ErrForbidden
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec("INSERT INTO posts(title,content,authorid) VALUES(?,?,?)", title, content, user.ID)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	for _, category := range categories {
		if _, err = tx.Exec("INSERT INTO postsandcats(threadid,postid) VALUES(?,?)", category, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) Delete(user User, kind string, id int64) error {
	if user.Role != "admin" {
		return ErrForbidden
	}
	switch kind {
	case "post":
		return affected(s.DB.Exec("DELETE FROM posts WHERE id=?", id))
	case "comment":
		return affected(s.DB.Exec("DELETE FROM comments WHERE id=?", id))
	case "thread":
		return affected(s.DB.Exec("DELETE FROM threads WHERE id=?", id))
	}
	return fmt.Errorf("unknown deletion")
}
func (s *Store) AddThread(user User, title, description string) error {
	if user.Role != "admin" {
		return ErrForbidden
	}
	_, err := s.DB.Exec("INSERT INTO threads(title,description) VALUES(?,?)", title, description)
	return err
}
func (s *Store) AddComment(user User, post int64, content string) error {
	if user.ID == 0 {
		return ErrForbidden
	}
	_, err := s.DB.Exec("INSERT INTO comments(postid,user_id,content) VALUES(?,?,?)", post, user.ID, content)
	return err
}
func (s *Store) Vote(user User, comment bool, id int64, dislike bool) error {
	if user.ID == 0 {
		return ErrForbidden
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	voteType := "like"
	counter := "likes"
	if dislike {
		voteType = "dislike"
		counter = "dislikes"
	}
	table, actions, key := "posts", "rtactions", "postid"
	if comment {
		table, actions, key = "comments", "rtactionscomm", "commid"
	}
	// All identifiers are constants selected above; all values remain parameters.
	result, err := tx.Exec("INSERT INTO "+actions+"("+key+",user_id,type) VALUES(?,?,?) ON CONFLICT(user_id,"+key+") DO NOTHING", id, user.ID, voteType)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		if err = affected(tx.Exec("UPDATE "+table+" SET "+counter+"="+counter+"+1 WHERE id=?", id)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
