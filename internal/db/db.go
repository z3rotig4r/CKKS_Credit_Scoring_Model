package db

// MySQL Connection Management

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

func db() {
	db, err := sql.Open("mysql", "z3rotig4r:TKdlqj1@#@tcp(127.0.0.1:3306)/ckksCredit")
}
