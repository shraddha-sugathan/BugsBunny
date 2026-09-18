package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv" // Make sure to add this
)

var dbpool *pgxpool.Pool

func initDB() {
	// 1. Attempt to load the .env file explicitly
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or failed to load, relying on environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// 2. Added ?sslmode=disable to the fallback string
		dbURL = "postgres://postgres:postgres@localhost:5432/bugsbunny?sslmode=disable"
	}

	// 3. Connect to the pool
	dbpool, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	// 4. Actually test the connection (pgxpool.New doesn't ping by default)
	if err := dbpool.Ping(context.Background()); err != nil {
		log.Fatalf("Database ping failed (check credentials): %v\n", err)
	}

	log.Println("Connected to PostgreSQL successfully.")
}