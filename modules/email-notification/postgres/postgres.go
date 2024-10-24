package postgres

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	db *pgxpool.Pool
)

func ConnectDB(connString string) error {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return err
	}

	db = pool
	log.Println("Connected to PostgreSQL")
	return nil
}

func CloseDB() {
	if db != nil {
		db.Close()
		log.Println("PostgreSQL connection closed")
	}
}
