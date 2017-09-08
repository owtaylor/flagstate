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
	fetchAllInterval, err := time.ParseDuration(config.Interval.FetchAll)
	if err != nil {
		log.Fatalf("Can't parse interval.fetch_all (%s) as a duration: %s", config.Interval.FetchAll, err)
	}
	gcInterval, err := time.ParseDuration(config.Interval.GarbageCollect)
	if err != nil {
		log.Fatalf("Can't parse interval.garbage_collect (%s) as a duration: %s", config.Interval.GarbageCollect, err)
	}
	go func() {
		fetchAllTicker := time.Tick(fetchAllInterval)
		gcTicker := time.Tick(gcInterval)

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

	http.Handle("/events", NewEventHandler(config, fetcher))
	http.Handle("/assert", NewAssertHandler(db, changes))
	http.Handle("/index", NewIndexHandler(db))
	http.Handle("/", NewHomeHandler(db))

	log.Fatal(http.ListenAndServe(":8088", handlers.LoggingHandler(os.Stdout, http.DefaultServeMux)))
}
