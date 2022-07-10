package database

import (
	"embed"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

var (
	//go:embed migrations/*.sql
	migrationsDir     embed.FS
	migrationsDirName = "migrations"
)

func ConnectToDatabase(driver, dsn string) *sqlx.DB {
	if dsn == "" {
		return nil
	}
	con, err := sqlx.Connect(driver, dsn)
	if err != nil {
		log.Fatalf("establish db connection, %v", err)
	}

	con.SetMaxOpenConns(20)
	con.SetMaxIdleConns(20)
	con.SetConnMaxIdleTime(time.Second * 30)
	con.SetConnMaxLifetime(time.Minute * 2)

	migrateDatabase(con)
	return con
}

func migrateDatabase(db *sqlx.DB) {
	goose.SetBaseFS(migrationsDir)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("migrate db: %v", err)
	}

	if err := goose.Up(db.DB, migrationsDirName); err != nil {
		log.Fatalf("migrate db up: %v", err)
	}
}
