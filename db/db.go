package db

import (
	"context"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/TeamTenuki/twiddler/config"
)

var db *sqlx.DB

// Init initialises the DB connection, so that is now possible
// to put it into context using NewContext function.
// The DB file will be placed at given path, or, if path is an empty string,
// the default path will be chosen with the name "twiddler.db" placed at
// default config directory (see config.Dir).
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

// MustInit calls Init and panics on errors.
func MustInit(path string) {
	if err := Init(path); err != nil {
		panic(err)
	}
}

type contextKey int

const (
	dbContextKey contextKey = iota
)

// NewContext returns a context with DB connection in it.
// The connection is retrievable with FromContext function.
func NewContext(c context.Context) context.Context {
	return context.WithValue(c, dbContextKey, db)
}

// FromContext retrieves a DB connection from context.
// Function will crash if supplied context wasn't created by NewContext.
func FromContext(c context.Context) *sqlx.DB {
	return c.Value(dbContextKey).(*sqlx.DB)
}
