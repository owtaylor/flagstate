package web

import (
	"context"
	"github.com/owtaylor/flagstate"
	"github.com/owtaylor/flagstate/database"
	"html/template"
	"log"
	"net/http"
)

type homeHandler struct {
	config *flagstate.Config
	db     database.Database
}

var homeTemplate *template.Template

func init() {
	var err error
	homeTemplate, err = template.New("foo").Parse(repositoriesHtmlTemplate)
	if err != nil {
		panic(err)
	}
}

func (hh *homeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	SetCacheControl(w, hh.config.Cache.MaxAgeHtml.Value, false)
	if CheckAndSetETag(hh.db, w, r) {
		return
	}

	ctx := context.Background()
	results, err := hh.db.DoQuery(ctx, database.NewQuery())
	if err != nil {
		internalError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	err = homeTemplate.Execute(w, results)
	if err != nil {
		log.Print(err)
	}
}
