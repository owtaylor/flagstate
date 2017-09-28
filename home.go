package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

type homeHandler struct {
	db       Database
	maxAge   int
	template *template.Template
}

func NewHomeHandler(config *Config, db Database) http.Handler {
	hh := &homeHandler{
		db:     db,
		maxAge: int(0.5 + config.Cache.MaxAgeHtml.Value.Seconds()),
	}

	var err error
	hh.template, err = template.New("foo").Parse(repositoriesHtmlTemplate)
	if err != nil {
		log.Fatal(err)
	}

	return hh
}

func (hh *homeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%v", hh.maxAge))
	if CheckAndSetETag(hh.db, w, r) {
		return
	}

	ctx := context.Background()
	results, err := hh.db.DoQuery(ctx, NewQuery())
	if err != nil {
		internalError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	err = hh.template.Execute(w, results)
	if err != nil {
		log.Print(err)
	}
}
