package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/TeamTenuki/twiddler"
	"github.com/TeamTenuki/twiddler/config"
	"github.com/TeamTenuki/twiddler/db"
)

var cmdline struct {
	config string
	db     string
}

func main() {
	flag.StringVar(&cmdline.config, "config", "", "Path to a configuration file containing API keys.")
	flag.StringVar(&cmdline.db, "db", "", "Path to a SQLite DB file to persist data.")
	flag.Parse()

	if err := db.Init(cmdline.db); err != nil {
		log.Fatalf("ERROR: failed to initialise DB: %s", err)
	}

	config, err := config.Parse(cmdline.config)
	if err != nil {
		log.Fatalf("ERROR: failed to parse config file: %s", err)
	}

	c := db.NewContext(context.Background())
	c = withSignalCancel(c)

	if err := twiddler.Run(c, config); err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}

func withSignalCancel(c context.Context) context.Context {
	c, cancel := context.WithCancel(c)

	go func() {
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

		<-sc

		cancel()
	}()

	return c
}
