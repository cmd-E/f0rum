//go:build cgo

package forum

import (
	"database/sql"
	"path/filepath"
	"testing"
)

const legacyFixture = `
CREATE TABLE users(id INTEGER PRIMARY KEY, username TEXT, email TEXT, password TEXT);
CREATE TABLE threads(id INTEGER PRIMARY KEY, title TEXT, description TEXT);
CREATE TABLE posts(id INTEGER PRIMARY KEY, title TEXT, content TEXT, likes INTEGER, dislikes INTEGER, author TEXT, authorid INTEGER);
CREATE TABLE postsandcats(id INTEGER PRIMARY KEY, threadid INTEGER, postid INTEGER);
CREATE TABLE comments(id INTEGER PRIMARY KEY, postid INTEGER, username TEXT, content TEXT, likes INTEGER, dislikes INTEGER);
CREATE TABLE rtactions(id INTEGER PRIMARY KEY, postid INTEGER, username TEXT, userid INTEGER, type TEXT);
CREATE TABLE rtactionscomm(id INTEGER PRIMARY KEY, commid INTEGER, username TEXT);
CREATE TABLE sessions(session TEXT, username TEXT);
INSERT INTO users VALUES(7,'legacy','legacy@example.invalid','hash');
INSERT INTO threads VALUES(3,'thread','description');
INSERT INTO posts VALUES(2,'title','body',1,0,'legacy',7);
INSERT INTO postsandcats VALUES(1,3,2);
INSERT INTO comments VALUES(4,2,'legacy','comment',0,0);
INSERT INTO rtactions VALUES(1,2,'legacy',7,'like');
INSERT INTO rtactionscomm VALUES(1,4,'legacy');
INSERT INTO sessions VALUES('insecure-token','legacy');
`

func legacyStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "legacy.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.DB.Close() })
	if _, err := s.DB.Exec(legacyFixture); err != nil {
		t.Fatal(err)
	}
	return s, path
}

func TestLegacyMigrationPreservesIdentityAndBacksUpBeforeWriting(t *testing.T) {
	s, path := legacyStore(t)
	if err := s.Initialize(); err == nil {
		t.Fatal("legacy database initialized without explicit migration")
	}
	if err := s.Migrate(path, 7); err != nil {
		t.Fatal(err)
	}
	user, err := s.User(7)
	if err != nil || user.Role != "admin" || user.Name != "legacy" {
		t.Fatal(user, err)
	}
	var author, sessions int
	if err := s.DB.QueryRow("SELECT authorid FROM posts WHERE id=2").Scan(&author); err != nil || author != 7 {
		t.Fatal(author, err)
	}
	if err := s.DB.QueryRow("SELECT count(*) FROM sessions").Scan(&sessions); err != nil || sessions != 0 {
		t.Fatal(sessions, err)
	}
	var vote sql.NullString
	if err := s.DB.QueryRow("SELECT type FROM rtactionscomm WHERE id=1").Scan(&vote); err != nil || vote.Valid {
		t.Fatal("legacy vote direction invented", vote, err)
	}
	backups, err := filepath.Glob(path + ".backup-*")
	if err != nil || len(backups) != 1 {
		t.Fatal(backups, err)
	}
	backup, err := sql.Open("sqlite3", backups[0]+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	var version int
	if err := backup.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 0 {
		t.Fatal("backup is not the original schema", version, err)
	}
	if err := backup.QueryRow("SELECT count(*) FROM sessions WHERE session='insecure-token'").Scan(&sessions); err != nil || sessions != 1 {
		t.Fatal("backup data changed", sessions, err)
	}
	if err := s.Migrate(path, 0); err != nil {
		t.Fatal("idempotent migration", err)
	}
}

func TestConflictingLegacyMigrationRollsBackAndPreservesBackup(t *testing.T) {
	for _, statement := range []string{
		"UPDATE posts SET authorid=999",
		"INSERT INTO users VALUES(8,'legacy','other@example.invalid','hash')",
	} {
		t.Run(statement, func(t *testing.T) {
			s, path := legacyStore(t)
			if _, err := s.DB.Exec(statement); err != nil {
				t.Fatal(err)
			}
			if err := s.Migrate(path, 0); err == nil {
				t.Fatal("conflicting migration accepted")
			}
			if version, err := s.Version(); err != nil || version != 0 {
				t.Fatal(version, err)
			}
			var temporaryTables int
			if err := s.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE name LIKE '%_new'").Scan(&temporaryTables); err != nil || temporaryTables != 0 {
				t.Fatal("partial schema persisted", temporaryTables, err)
			}
			var title string
			if err := s.DB.QueryRow("SELECT title FROM posts WHERE id=2").Scan(&title); err != nil || title != "title" {
				t.Fatal("original content lost", title, err)
			}
			backups, err := filepath.Glob(path + ".backup-*")
			if err != nil || len(backups) != 1 {
				t.Fatal("missing backup", backups, err)
			}
		})
	}
}
