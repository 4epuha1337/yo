package db

import (
	"log"
	"os"
	"path/filepath"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

const (
	SQLCreateScheduler = `
	CREATE TABLE scheduler (
	    id      INTEGER PRIMARY KEY, 
	    date    CHAR(8) NOT NULL DEFAULT "", 
	    title   TEXT NOT NULL DEFAULT "",
		comment TEXT NOT NULL DEFAULT "",
		repeat  VARCHAR(128) NOT NULL DEFAULT "" 
	);
	`
	SQLCreateSchedulerIndex = `
	CREATE INDEX scheduler_date_index ON scheduler (date)
	`
)

var sqlDB *sql.DB

func checkDB() (*sql.DB, error) {
	appPath, err := os.Executable()
	if err != nil {
	    log.Fatal(err)
	}
	dbFile := filepath.Join(filepath.Dir(appPath), "scheduler.db")
	_, err = os.Stat(dbFile)

	var install bool
	if err != nil {
 	   install = true
	}
	
	if install {
		_, err = os.Create(dbFile)
		if err != nil {
			return nil, err
		}
		sqlDB, err := sql.Open("sqlite3", dbFile)
		if err != nil {
			return nil, err
		}

		if _, err = sqlDB.Exec(SQLCreateScheduler); err != nil {
			return nil, err
		}
	}
	return sqlDB, nil
}