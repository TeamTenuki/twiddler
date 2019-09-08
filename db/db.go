package db

import (
	"context"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/TeamTenuki/twiddler/config"
)

var db *sqlx.DB

func Init(path string) (err error) {
	dbFilepath := path

	if path == "" {
		configDir, err := config.Dir()
		if err != nil {
			return err
		}

		dbFilepath = filepath.Join(configDir, "twiddler.db")
	}

	db, err = sqlx.Open("sqlite3", dbFilepath)

	return err
}

func MustOpen(path string) {
	if err := Init(path); err != nil {
		panic(err)
	}
}

type contextKey int

const (
	dbContextKey contextKey = iota
)

func NewContext(c context.Context) context.Context {
	return context.WithValue(c, dbContextKey, db)
}

func FromContext(c context.Context) *sqlx.DB {
	return c.Value(dbContextKey).(*sqlx.DB)
}
