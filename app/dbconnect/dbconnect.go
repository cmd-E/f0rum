package dbconnect

import (
	"database/sql"
	"log"
)

// DBConn global var for sqlite db. Opens ones at sturt up and uses evrywhere
var DBConn *sql.DB

// ConnectdbFunc connects to the forum.db and returns database pointer
func ConnectdbFunc() *sql.DB {
	database, err := sql.Open("sqlite3", "file:../database/forum.db?cache=shared")
	if err != nil {
		log.Fatalf("Open db error: %s", err.Error())
	}
	//database.SetMaxOpenConns(1)
	//Database.SetMaxOpenConns(2) // Достаточно для того, чтобы проверять пользователей и сессию
	return database
}
