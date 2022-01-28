package driver

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"time"
)

type DB struct {
	SQL *sql.DB
}

var dbConn = &DB{}

const maxOpenDbConn = 25
const maxIdleDbConn = 25
const maxDbLifetime = 5 * time.Minute

func ConnectMysql(dsn string) (*DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(maxOpenDbConn)
	db.SetMaxIdleConns(maxIdleDbConn)
	db.SetConnMaxLifetime(maxDbLifetime)
	dbConn.SQL = db

	err = testDB(err, db)

	return dbConn, err
}

func testDB(err error, db *sql.DB) error {
	err = db.Ping()
	if err != nil {
		fmt.Println("Error generated", err)
	} else {
		log.Println("=== Pinged database successfully! ===")
	}
	return err
}
