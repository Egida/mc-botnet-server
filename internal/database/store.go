package database

import "github.com/stephenafamo/bob"

type Store interface {
}

type SQLStore struct {
	db bob.DB
}

func NewSQLStore(db bob.DB) *SQLStore {
	return &SQLStore{db}
}
