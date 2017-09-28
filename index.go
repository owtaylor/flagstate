package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type indexHandler struct {
	config  *Config
	db      Database
	dynamic bool
}

func (ih *indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Registry string
		Results  []*Repository
	}

	if ih.config.Registry.PublicUrl != "" {
		body.Registry = ih.config.Registry.PublicUrl
	} else {
		body.Registry = ih.config.Registry.Url
	}

	SetCacheControl(w, ih.config.Cache.MaxAgeIndex.Value, ih.dynamic)
	if CheckAndSetETag(ih.db, w, r) {
		return
	}

	r.ParseForm()
	q := NewQuery()

	for k, v := range r.Form {
		v0 := v[0]
		switch k {
		case "repository":
			q.Repository(v0)
		case "tag":
			q.Tag(v0)
		case "os":
			q.OS(v0)
		case "arch":
			q.Arch(v0)
		default:
			is_annotation := false
			if strings.HasPrefix(k, "annotation:") {
				k = strings.TrimPrefix(k, "annotation:")
				is_annotation = true
			} else if strings.HasPrefix(k, "label:") {
				is_annotation = true
			}
			if is_annotation {
				if strings.HasSuffix(k, ":exists") {
					k = strings.TrimSuffix(k, ":exists")
					switch strings.ToLower(v0) {
					case "true", "1":
						q.AnnotationExists(k)
					}
				} else {
					q.AnnotationIs(k, v0)
				}
			}
		}
	}

	ctx := context.Background()

	var err error
	body.Results, err = ih.db.DoQuery(ctx, q)
	if err != nil {
		internalError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	err = encoder.Encode(body)
	if err != nil {
		log.Print(err)
	}
}
