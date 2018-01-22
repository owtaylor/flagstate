package web

import (
	"fmt"
	"github.com/owtaylor/flagstate"
	"github.com/owtaylor/flagstate/database"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var tokenRegexp = regexp.MustCompile(
	`^(\*|(?:W/)?"(?:[^"\\]+|\\.)*")` + // `*` or `"foo"` or `W/"foo"`
		`(?:\s*,\s*|$)` + // followed by a comma or the end
		`|` +
		`^\s*,\s*`) // OR just a comma (null elements are allowed but don't count)

func ParseIfMatch(input string) ([]string, error) {
	result := make([]string, 0)
	s := strings.TrimSpace(input)
	for {
		match := tokenRegexp.FindStringSubmatchIndex(s)
		if match == nil {
			return nil, fmt.Errorf("Error parsing If-[Not-]Match header %v at %v", input, s)
		}
		if match[2] != -1 {
			result = append(result, s[match[2]:match[3]])
		}
		if match[1] == len(s) {
			break
		}
		s = s[match[1]:]
	}

	return result, nil
}

func SetCacheControl(w http.ResponseWriter, maxAge time.Duration, dynamic bool) {
	var value string
	if dynamic {
		// This may be too drastic - it could turn out that the same
		// "dynamic" queries are being made over and over again, and
		// caching is useful. We could make setting this configurable
		// and let front-ends determine if they want to use different
		// policies for /index/dynamic and /index/static.
		value = "no-store"
	} else {
		value = fmt.Sprintf("max-age=%v", int(0.5+maxAge.Seconds()))
	}
	w.Header().Set("Cache-Control", value)
}

func CheckAndSetETag(db database.Database, w http.ResponseWriter, r *http.Request) bool {
	modificationTime, err := db.ModificationTime()
	if err != nil {
		internalError(w, err)
		return true
	}

	// We use the modification time for an ETag rather than Last-Modified,
	// because of the limited (1-sec) resolution of Last-Modified.
	// With Last-Modified, there's a reasonable chance of stale data being
	// cached if a response is cached at the wrong time.
	//
	// Embedding the BuildId into the ETag is super useful during development.
	// In production it would cause cache misses during a rolling deploypment,
	// but still would have the intended effect of avoiding validation of
	// data generated with the old build.
	etag := `"` + flagstate.BuildId + "-" + modificationTime.Format(time.RFC3339Nano) + `"`

	for _, val := range r.Header["If-None-Match"] {
		candidates, err := ParseIfMatch(val)
		if err != nil {
			badRequest(w, err)
			return true
		}
		for _, c := range candidates {
			if c == etag || c == "*" {
				w.Header().Set("ETag", etag)
				w.WriteHeader(http.StatusNotModified)
				return true
			}
		}
	}

	w.Header().Set("ETag", etag)

	return false
}

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
