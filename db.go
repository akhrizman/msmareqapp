package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func initDB() {
	// read DSN from env or default
	// DSN format: username:password@tcp(host:port)/msmareqdb?parseTime=true&charset=utf8mb4
	dsn := os.Getenv("MSMAREQ_DSN")
	if dsn == "" {
		dsn = "msmarequser:yourpassword@tcp(127.0.0.1:3306)/msmareqdb?parseTime=true&charset=utf8mb4"
	}

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("db open error: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("db ping error: %v", err)
	}
	fmt.Println("Connected to DB")
}
