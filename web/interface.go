package web

import (
	"github.com/gorilla/handlers"
	"github.com/owtaylor/flagstate"
	"github.com/owtaylor/flagstate/database"
	"github.com/owtaylor/flagstate/fetcher"
	"github.com/owtaylor/flagstate/util"
	"log"
	"net/http"
	"os"
)

type WebInterface struct {
	Config  *flagstate.Config
	DB      database.Database
	Fetcher *fetcher.Fetcher
	Changes *util.ChangeBroadcaster
}

func (wi *WebInterface) Start() {
	http.Handle("/events", &eventHandler{
		config:  wi.Config,
		fetcher: wi.Fetcher,
	})
	if wi.Config.Components.AssertEndpoint {
		http.Handle("/assert", &assertHandler{
			db:      wi.DB,
			changes: wi.Changes,
		})
	}
	http.Handle("/index/static", &indexHandler{
		config: wi.Config,
		db:     wi.DB,
	})
	http.Handle("/index/dynamic", &indexHandler{
		config:  wi.Config,
		db:      wi.DB,
		dynamic: true,
	})
	if wi.Config.Components.WebUI {
		http.Handle("/", &homeHandler{
			config: wi.Config,
			db:     wi.DB,
		})
	}

	log.Fatal(http.ListenAndServe(":8088", handlers.LoggingHandler(os.Stdout, http.DefaultServeMux)))
}
