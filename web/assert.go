package web

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/owtaylor/flagstate"
	"github.com/owtaylor/flagstate/database"
	"github.com/owtaylor/flagstate/util"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	AssertionContains    = "contains"
	AssertionNotContains = "not-contains"
)

type Assertion struct {
	Type string
	Test interface{}
}

type Failure struct {
	Assertion
}

type assertHandler struct {
	db      database.Database
	changes *util.ChangeBroadcaster
}

func jsonContains(a interface{}, b interface{}) bool {
	switch a_v := a.(type) {
	case map[string]interface{}:
		b_v, ok := b.(map[string]interface{})
		if !ok {
			return false
		}
		for key, b_value := range b_v {
			a_value, ok := a_v[key]
			if !ok {
				return false
			}
			if !jsonContains(a_value, b_value) {
				return false
			}
		}
	case []interface{}:
		b_v, ok := b.([]interface{})
		if !ok {
			return false
		}
		for _, b_value := range b_v {
			found := false
			for _, a_value := range a_v {
				if jsonContains(a_value, b_value) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	case int, string, bool:
		if a != b {
			return false
		}
	default:
		return false
	}

	return true
}

func checkAssertions(result interface{}, assertions []Assertion) (failures []Failure, err error) {
	for _, assertion := range assertions {
		switch assertion.Type {
		case AssertionContains:
			if !jsonContains(result, assertion.Test) {
				failures = append(failures, Failure{
					Assertion: assertion,
				})
			}
		case AssertionNotContains:
			if jsonContains(result, assertion.Test) {
				failures = append(failures, Failure{
					Assertion: assertion,
				})
			}
		default:
			err = fmt.Errorf("Unknown assertion type %s", assertion.Type)
			return
		}
	}

	return
}

func (ah *assertHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Query_     database.Query `json:"Query"`
		Assertions []Assertion
	}

	var reply struct {
		Success  bool
		Results  []*flagstate.Repository
		Failures []Failure
	}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&body)
	if err != nil {
		badRequest(w, err)
		return
	}

	timeout := 0
	if s := r.URL.Query().Get("timeout"); s != "" {
		timeout, err = strconv.Atoi(s)
		if err != nil {
			badRequest(w, fmt.Errorf("Cannot parse timeout: %v", err))
		}
	}

	lastChange := ah.changes.LastChange()
	expires := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		ctx := context.Background()
		reply.Results, err = ah.db.DoQuery(ctx, &body.Query_)
		if err != nil {
			internalError(w, err)
			return
		}

		resultJson, err := json.Marshal(reply.Results)
		if err != nil {
			internalError(w, err)
			return
		}

		var resultGeneric interface{}
		err = json.Unmarshal(resultJson, &resultGeneric)
		if err != nil {
			internalError(w, err)
			return
		}

		reply.Failures, err = checkAssertions(resultGeneric, body.Assertions)
		if err != nil {
			badRequest(w, err)
			return
		}

		reply.Success = len(reply.Failures) == 0
		if reply.Success {
			break
		}

		ok := false
		if timeout != 0 {
			lastChange, ok = ah.changes.WaitTimeout(lastChange, time.Until(expires))
		}

		if !ok {
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if reply.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		// It's not a BadRequest in HTTP terms, but having a non-Success
		// failure code makes it easier to script against the interface
		w.WriteHeader(http.StatusBadRequest)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "    ")
	err = encoder.Encode(reply)
	if err != nil {
		log.Print(err)
	}
}
