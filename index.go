package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

type indexHandler struct {
	db Database
}

func NewIndexHandler(db Database) http.Handler {
	return &indexHandler{
		db: db,
	}
}

func (ih *indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	results, err := ih.db.DoQuery(ctx, NewQuery())
	if err != nil {
		internalError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	err = encoder.Encode(results)
	if err != nil {
		log.Print(err)
	}
}
