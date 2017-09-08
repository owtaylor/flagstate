package main

import (
	"encoding/json"
	"fmt"
	"github.com/docker/distribution/notifications"
	"net/http"
	"strings"
)

type eventHandler struct {
	config  *Config
	fetcher *Fetcher
}

func NewEventHandler(config *Config, fetcher *Fetcher) http.Handler {
	return &eventHandler{
		config:  config,
		fetcher: fetcher,
	}
}

func (eh *eventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		X      int
		Events []notifications.Event
	}

	if r.Method != "POST" {
		badRequest(w, fmt.Errorf("Can only POST to /events"))
		return
	}

	if eh.config.Events.Token != "" {
		authorized := false
		for _, header := range r.Header["Authorization"] {
			fields := strings.Fields(header)
			if fields[0] == "Bearer" && fields[1] == eh.config.Events.Token {
				authorized = true
			}
		}
		if !authorized {
			w.Header().Set("Content-Type", "text/plain")
			// We violate the HTTP spec by not including WWW-Authenticate, but
			// there's nothing meaningful to provide
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "No/incorrect authorization token provided\n")
			return
		}
	}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&body)
	if err != nil {
		badRequest(w, err)
		return
	}

	for _, event := range body.Events {
		switch event.Action {
		case notifications.EventActionPush, notifications.EventActionDelete:
			eh.fetcher.FetchRepository(event.Target.Repository)
			break
		}
	}
}
