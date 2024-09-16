package main

import (
	"errors"
	"flag"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var postgresURL, migrationsPath string

	flag.StringVar(&postgresURL, "postgresURL", "", "path to storage")
	flag.StringVar(&migrationsPath, "migrations", "", "path to migrations")
	flag.Parse()

	m, err := migrate.New(
		"file://"+migrationsPath,
		postgresURL,
	)
	if err != nil {
		panic(err)
	}
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("no migrations to apply")
			return
		}

		panic(err)
	}
}
