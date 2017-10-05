package main

import (
	"encoding/json"
	"strconv"
)

type whereBuilder struct {
	pieces []string
	args   []interface{}
}

func (wb *whereBuilder) flatten() string {
	result := ""
	i := 0
	in_or := false
	n_ands := 0
	n_ors := 0
	for i = 0; i < len(wb.pieces); i++ {
		if wb.pieces[i] == "" {
			if in_or {
				if n_ors > 1 {
					result += ")"
				}
				in_or = false
				n_ors = 0
			}
		} else {
			if !in_or {
				if n_ands > 0 {
					result += " AND "
				}
				if i+1 < len(wb.pieces) && wb.pieces[i+1] != "" {
					result += "("
				}
				n_ands++
				in_or = true
			} else {
				if n_ors > 0 {
					result += " OR "
				}
			}
			n_ors++
			result += wb.pieces[i]
		}
	}

	if n_ors > 1 {
		result += ")"
	}

	return result
}

func likePattern(globPattern string) string {
	pattern := ""
	for _, c := range globPattern {
		switch c {
		case '%', '_':
			pattern += "\\" + string(c)
		case '*':
			pattern += "%"
		case '?':
			pattern += "_"
		default:
			pattern += string(c)
		}
	}

	return pattern
}

func (wb *whereBuilder) addArg(arg interface{}) string {
	wb.args = append(wb.args, arg)
	return `$` + strconv.Itoa(len(wb.args))
}

func (wb *whereBuilder) addPiece(piece string) {
	wb.pieces = append(wb.pieces, piece)
}

func (wb *whereBuilder) makeWhereSubclause(subject string, terms []QueryTerm) {
	for _, term := range terms {
		switch term.queryType {
		case QueryIs:
			wb.addPiece(subject + ` = ` + wb.addArg(term.argument))
		case QueryMatches:
			wb.addPiece(subject + ` like ` + wb.addArg(likePattern(term.argument)))
		case QueryExists:
			panic("QueryExists cannot be handled generically")
		}
	}
	wb.addPiece("")
}

func (wb *whereBuilder) makeAnnotationSubclause(annotation string, terms []QueryTerm) {
	for _, term := range terms {
		switch term.queryType {
		case QueryIs:
			wb.addPiece(`i.annotations ? ` + wb.addArg(annotation))
		case QueryMatches:
			wb.addPiece(`jsonb_object_field_text(i.annotations, ` + wb.addArg(annotation) + `) ` +
				`like ` + wb.addArg(likePattern(term.argument)))
		case QueryExists:
			argJson, _ := json.Marshal(map[string]string{
				annotation: term.argument,
			})
			wb.addPiece(`i.annotations @> ` + wb.addArg(string(argJson)))
		}
	}
	wb.addPiece("")
}

func makeWhereClause(query *Query) (clause string, args []interface{}) {
	wb := whereBuilder{
		args:   make([]interface{}, 0, 20),
		pieces: make([]string, 0, 20),
	}

	if len(query.repository) > 0 {
		wb.makeWhereSubclause(`t.repository`, query.repository)
	}

	if len(query.tag) > 0 {
		wb.makeWhereSubclause(`t.tag`, query.tag)
	}

	if len(query.os) > 0 {
		wb.makeWhereSubclause(`i.os`, query.os)
	}

	if len(query.arch) > 0 {
		wb.makeWhereSubclause(`i.arch`, query.arch)
	}

	for annotation, terms := range query.annotations {
		wb.makeAnnotationSubclause(annotation, terms)
	}

	args = wb.args
	if len(wb.pieces) > 0 {
		clause = ` WHERE ` + wb.flatten()
	}

	return
}
