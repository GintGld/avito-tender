package app

import (
	postgres "tender/internal/storage/postgres"
)

type Storage struct {
	Postgres *postgres.Storage
}

func New(connURL string) (*Storage, error) {
	postgres, err := postgres.New(connURL)
	if err != nil {
		return nil, err
	}

	return &Storage{postgres}, nil
}
