package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

type DB struct {
	sql    *sql.DB
	buffer []Restart
}

type Restart struct {
	Server  string
	Command string
	Time    uint16 // Amount of minutes from day start
}

const (
	schemaSQL = `
	CREATE TABLE IF NOT EXISTS restart (
		server VARCHAR(32) PRIMARY KEY,
		command VARCHAR(32),
		time INTEGER
	);`

	insertSQL = `
	INSERT INTO restart (
		server, command, time
	) VALUES (
		?, ?, ?
	);`

	selectSql = "SELECT server, command, time FROM restart;"

	clearSql = "DELETE FROM restart;"
)

func NewDB(dbFile string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	if _, err = sqlDB.Exec(schemaSQL); err != nil {
		return nil, err
	}

	rows, err := sqlDB.Query(selectSql)
	if !(err == nil || strings.Contains(err.Error(), "no rows")) {
		return nil, err
	}
	restarts := make([]Restart, 0, 5)
	for rows.Next() {
		restart := Restart{}
		err := rows.Scan(&restart.Server, &restart.Command, &restart.Time)
		if err != nil {
			log.Println(err)
			continue
		}

		restarts = append(restarts, restart)
	}
	rows.Close()
	log.Println("got restarts from db", restarts)

	if _, err := sqlDB.Exec(clearSql); err != nil {
		return nil, err
	}

	db := DB{
		sql:    sqlDB,
		buffer: restarts,
	}
	return &db, nil
}

func (db *DB) Add(restart Restart) error {
	if index := db.findIndex(restart.Server); index != -1 {
		existing := db.buffer[index]
		return fmt.Errorf("for server %s %s already is planned (old time: %q, new time: %q)", restart.Server, existing.Command, existing.Time, restart.Time)
	}

	db.buffer = append(db.buffer, restart)
	return nil
}

func (db *DB) findIndex(server string) int {
	for i, r := range db.buffer {
		if r.Server == server {
			return i
		}
	}

	return -1
}

func (db *DB) Select() []Restart {
	return db.buffer
}

func (db *DB) Delete(server string) error {
	if index := db.findIndex(server); index != -1 {
		db.buffer = slices.Delete(db.buffer, index, index+1)
		return nil
	}

	return errors.New("cannot find record" + server)
}

func (db *DB) Clear() {
	db.buffer = db.buffer[:0]
}

func (db *DB) Close() error {
	defer db.sql.Close()

	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}

	addStmt, err := db.sql.Prepare(insertSQL)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer addStmt.Close()

	for _, restart := range db.buffer {
		_, err := tx.Stmt(addStmt).Exec(restart.Server, restart.Command, restart.Time)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	db.buffer = db.buffer[:0]
	return tx.Commit()
}
