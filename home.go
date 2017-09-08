package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
)

type homeHandler struct {
	db       Database
	template *template.Template
}

func NewHomeHandler(db Database) http.Handler {
	hh := &homeHandler{
		db: db,
	}

	var err error
	hh.template, err = template.New("foo").Parse(repositoriesHtmlTemplate)
	if err != nil {
		log.Fatal(err)
	}

	return hh
}

func (hh *homeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
