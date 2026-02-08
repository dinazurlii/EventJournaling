package config

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func ConnectDB() {
	dsn := "postgres://asani@localhost:5432/postgres?sslmode=disable&search_path=event_journal"

	dbpool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err := dbpool.Ping(context.Background()); err != nil {
		log.Fatal("Database not reachable:", err)
	}

	DB = dbpool
	log.Println("✅ Database connected")
}
