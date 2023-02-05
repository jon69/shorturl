package dbh

import (
	"database/sql"
	"log"

	_ "github.com/jackc/pgx"
)

func Ping(conn string) bool {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		log.Print("Error connect to db: " + err.Error())
		return false
	}
	defer db.Close()
	return true
}
