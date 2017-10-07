package main

import (
	"context"
	"github.com/docker/distribution/digest"
	"time"
)

type QueryType int

const (
	QueryIs = iota
	QueryExists
	QueryMatches
)

type QueryTerm struct {
	queryType QueryType
	argument  string
}

type Query struct {
	repository   []QueryTerm
	tag          []QueryTerm
	os           []QueryTerm
	architecture []QueryTerm
	annotations  map[string][]QueryTerm
	labels       map[string][]QueryTerm
}

func NewQuery() *Query {
	return &Query{
		annotations: make(map[string][]QueryTerm),
		labels:      make(map[string][]QueryTerm),
	}
}

func (q *Query) Repository(repository string) *Query {
	q.repository = append(q.repository, QueryTerm{QueryIs, repository})
	return q
}

func (q *Query) Tag(tag string) *Query {
	q.tag = append(q.tag, QueryTerm{QueryIs, tag})
	return q
}

func (q *Query) TagMatches(tag string) *Query {
	q.tag = append(q.tag, QueryTerm{QueryMatches, tag})
	return q
}

func (q *Query) OS(os string) *Query {
	q.os = append(q.os, QueryTerm{QueryIs, os})
	return q
}

func (q *Query) Architecture(architecture string) *Query {
	q.architecture = append(q.architecture, QueryTerm{QueryIs, architecture})
	return q
}

func (q *Query) AnnotationExists(annotation string) *Query {
	q.annotations[annotation] = append(q.annotations[annotation],
		QueryTerm{QueryExists, ""})
	return q
}

func (q *Query) AnnotationIs(annotation string, value string) *Query {
	q.annotations[annotation] = append(q.annotations[annotation],
		QueryTerm{QueryIs, value})
	return q
}

func (q *Query) AnnotationMatches(annotation string, pattern string) *Query {
	q.annotations[annotation] = append(q.annotations[annotation],
		QueryTerm{QueryMatches, pattern})
	return q
}

func (q *Query) LabelExists(label string) *Query {
	q.labels[label] = append(q.labels[label],
		QueryTerm{QueryExists, ""})
	return q
}

func (q *Query) LabelIs(label string, value string) *Query {
	q.labels[label] = append(q.labels[label],
		QueryTerm{QueryIs, value})
	return q
}

func (q *Query) LabelMatches(label string, pattern string) *Query {
	q.labels[label] = append(q.labels[label],
		QueryTerm{QueryMatches, pattern})
	return q
}

type Tx interface {
	Commit() error
	Rollback() error
	Modified() (bool, time.Time)

	DoQuery(query *Query) ([]*Repository, error)

	StoreImage(repository string, image *TaggedImage) error
	StoreImageList(repository string, list *TaggedImageList) error

	SetImageTags(repository string, dgst digest.Digest, tags []string) error
	SetImageListTags(repository string, dgst digest.Digest, tags []string) error

	DeleteImage(repository string, dgst digest.Digest) error
	DeleteImageList(repository string, dgst digest.Digest) error

	DeleteMissingRepos(allRepos map[string]bool) error
	DeleteUnused() error
}

type Database interface {
	Begin(ctx context.Context) (Tx, error)
	// Convenience
	DoQuery(ctx context.Context, query *Query) ([]*Repository, error)

	ModificationTime() (time.Time, error)
}
