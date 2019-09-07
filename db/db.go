package db

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sqlx.DB

func GetConfigDirectory() string {
	homedir, exists := os.LookupEnv("HOME")
	if !exists {
		log.Fatalf("No home directory to store config at!")
	}

	configDir := filepath.Join(homedir, ".config", "twiddler")
	fi, err := os.Lstat(configDir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0777); err != nil {
			log.Fatalf("Can't create config dir: %s", err)
		}
		fi, err = os.Lstat(configDir)
	}

	if err != nil || !fi.IsDir() {
		log.Fatalf("Can't open a config directory (err: %s) (isDir: %t)", err, fi.IsDir())
	}

	return configDir
}

func init() {
	configDir := GetConfigDirectory()

	DB = sqlx.MustOpen("sqlite3", filepath.Join(configDir, "twiddler.db"))
}

type contextKey int

const (
	dbContextKey contextKey = iota
)

func NewContext(c context.Context) context.Context {
	return context.WithValue(c, dbContextKey, DB)
}

func FromContext(c context.Context) *sqlx.DB {
	return c.Value(dbContextKey).(*sqlx.DB)
}
