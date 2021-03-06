package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TeamTenuki/twiddler"
	"github.com/TeamTenuki/twiddler/config"
	"github.com/TeamTenuki/twiddler/db"
)

var cmdline struct {
	config  string
	db      string
	logTime bool
}

func main() {
	flag.StringVar(&cmdline.config, "config", "", "Path to a configuration file containing API keys.")
	flag.StringVar(&cmdline.db, "db", "", "Path to a SQLite DB file to persist data.")
	flag.BoolVar(&cmdline.logTime, "logTime", true, "Prepend date/time in the logger output.")
	flag.Parse()

	if !cmdline.logTime {
		log.SetFlags(0)
	}

	rand.Seed(time.Now().UnixNano())

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

		log.Println("Program shutdown...")

		cancel()
	}()

	return c
}
