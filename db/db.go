package db

import (
	"context"
	"log"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/TeamTenuki/twiddler/config"
)

var db *sqlx.DB

func init() {
	configDir, err := config.Dir()
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	db = sqlx.MustOpen("sqlite3", filepath.Join(configDir, "twiddler.db"))
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
