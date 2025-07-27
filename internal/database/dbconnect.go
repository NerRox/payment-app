package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	pgx_migrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

//go:embed migrations
var migrations embed.FS

func MustConnectPostgres() *pgxpool.Pool {
	var dbURL string = fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=disable", os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DATABASE_URL"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))

	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Opened connection for migrations")

	driver, err := pgx_migrate.WithInstance(db, &pgx_migrate.Config{})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Created dirver")

	d, err := iofs.New(migrations, "migrations")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Involved iofs")

	m, err := migrate.NewWithInstance("iofs", d, "postgres", driver)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Created migrator")

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migrated successfully")

	dbpool, err := pgxpool.New(context.Background(), dbURL)

	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}

	log.Println("Connected to DB with pgxpool")

	return dbpool
}
