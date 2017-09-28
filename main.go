package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/handlers"
	"log"
	"net/http"
	"os"
	"time"
)

var configFile = flag.String("config", "/etc/metastore/config.yaml", "Path to configuration file")

func internalError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusInternalServerError)
	log.Print(err)
	fmt.Fprintf(w, "Error: %v\n", err)
}

func badRequest(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusBadRequest)
	log.Print(err)
	fmt.Fprintf(w, "Error: %v\n", err)
}

func startTimers(config *Config, fetcher *Fetcher) {
	go func() {
		fetchAllTicker := time.Tick(config.Interval.FetchAll.Value)
		gcTicker := time.Tick(config.Interval.GarbageCollect.Value)

		for true {
			select {
			case <-fetchAllTicker:
				fetcher.FetchAll()
			case <-gcTicker:
				fetcher.GarbageCollect()
			}
		}
	}()
}

func main() {
	flag.Parse()

	config, err := LoadConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	var db Database

	if postgresUrl := config.Database.Postgres.Url; postgresUrl != "" {
		db, err = NewPostgresDB(postgresUrl)
		if err != nil {
			log.Fatal(err)
		}
	}

	if db == nil {
		log.Fatal("No database configured")
	}

	changes := NewChangeBroadcaster()
	fetcher := NewFetcher(db, changes, config.Registry.Url)
	fetcher.FetchAll()
	startTimers(config, fetcher)

	http.Handle("/events", &eventHandler{
		config:  config,
		fetcher: fetcher,
	})
	http.Handle("/assert", &assertHandler{
		db:      db,
		changes: changes,
	})
	http.Handle("/index/static", &indexHandler{
		config: config,
		db:     db,
	})
	http.Handle("/index/dynamic", &indexHandler{
		config:  config,
		db:      db,
		dynamic: true,
	})
	http.Handle("/", &homeHandler{
		config: config,
		db:     db,
	})

	log.Printf("metastore: %s", BuildString)
	log.Fatal(http.ListenAndServe(":8088", handlers.LoggingHandler(os.Stdout, http.DefaultServeMux)))
}
