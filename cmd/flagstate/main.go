package main

import (
	"flag"
	"github.com/owtaylor/flagstate"
	"github.com/owtaylor/flagstate/database"
	"github.com/owtaylor/flagstate/fetcher"
	"github.com/owtaylor/flagstate/util"
	"github.com/owtaylor/flagstate/web"
	"log"
	"time"
)

var configFile = flag.String("config", "/etc/flagstate/config.yaml", "Path to configuration file")

func startTimers(config *flagstate.Config, fetcher *fetcher.Fetcher) {
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

	config, err := flagstate.LoadConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	var db database.Database

	if postgresUrl := config.Database.Postgres.Url; postgresUrl != "" {
		db, err = database.NewPostgresDB(postgresUrl)
		if err != nil {
			log.Fatal(err)
		}
	}

	if db == nil {
		log.Fatal("No database configured")
	}

	changes := util.NewChangeBroadcaster()
	fetcher := fetcher.NewFetcher(db, changes, config.Registry.Url)
	fetcher.FetchAll()
	startTimers(config, fetcher)

	web := &web.WebInterface{
		Config:  config,
		DB:      db,
		Fetcher: fetcher,
		Changes: changes,
	}

	log.Printf("flagstate: %s", flagstate.BuildString)
	web.Start()
}
