package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	tables := []string{"temperatures", "humidities", "co2s", "smells"}
	for _, table := range tables {
		res, err := db.Exec("DELETE FROM "+table+" WHERE recorded_at < NOW() - INTERVAL 7 DAY")
		if err != nil {
			log.Fatalf("failed to delete from %s: %v", table, err)
		}
		n, _ := res.RowsAffected()
		log.Printf("deleted %d rows from %s", n, table)
	}
}
