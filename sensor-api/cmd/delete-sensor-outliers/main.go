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

	for _, table := range []string{"humidities", "temperatures"} {
		deleteOutliers(db, table)
	}
}

func deleteOutliers(db *sql.DB, table string) {
	rows, err := db.Query(`
		SELECT h.id, h.sensor_id, h.value, h.recorded_at
		FROM `+table+` h
		JOIN (
			SELECT sensor_id, AVG(value) AS avg_val, STDDEV(value) AS std_val
			FROM `+table+`
			WHERE recorded_at >= NOW() - INTERVAL 1 HOUR
			GROUP BY sensor_id
		) stats ON h.sensor_id = stats.sensor_id
		WHERE h.recorded_at >= NOW() - INTERVAL 1 HOUR
		AND (h.value < stats.avg_val - 3 * stats.std_val
			OR h.value > stats.avg_val + 3 * stats.std_val)
	`)
	if err != nil {
		log.Fatalf("failed to query outliers from %s: %v", table, err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		var sensorID, recordedAt, value string
		if err := rows.Scan(&id, &sensorID, &value, &recordedAt); err != nil {
			log.Fatal(err)
		}
		log.Printf("outlier: table=%s id=%d sensor_id=%s value=%s recorded_at=%s", table, id, sensorID, value, recordedAt)
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		log.Printf("no outliers found in %s", table)
		return
	}

	res, err := db.Exec(`
		DELETE FROM ` + table + `
		WHERE recorded_at >= NOW() - INTERVAL 1 HOUR
		AND id IN (
			SELECT id FROM (
				SELECT h.id
				FROM ` + table + ` h
				JOIN (
					SELECT sensor_id, AVG(value) AS avg_val, STDDEV(value) AS std_val
					FROM ` + table + `
					WHERE recorded_at >= NOW() - INTERVAL 1 HOUR
					GROUP BY sensor_id
				) stats ON h.sensor_id = stats.sensor_id
				WHERE h.recorded_at >= NOW() - INTERVAL 1 HOUR
				AND (h.value < stats.avg_val - 3 * stats.std_val
					OR h.value > stats.avg_val + 3 * stats.std_val)
			) tmp
		)
	`)
	if err != nil {
		log.Fatalf("failed to delete outliers from %s: %v", table, err)
	}
	n, _ := res.RowsAffected()
	log.Printf("deleted %d outlier rows from %s", n, table)
}
