package main

import (
	"context"
	"github.com/docker/distribution/digest"
	"time"
)

type Query struct {
	repository  string
	tag         string
	os          string
	arch        string
	annotations map[string]string
}

func NewQuery() *Query {
	return &Query{
		annotations: make(map[string]string),
	}
}

func (q *Query) Repository(repository string) *Query {
	q.repository = repository
	return q
}

func (q *Query) Tag(tag string) *Query {
	q.tag = tag
	return q
}

func (q *Query) OS(os string) *Query {
	q.os = os
	return q
}

func (q *Query) Arch(arch string) *Query {
	q.arch = arch
	return q
}

func (q *Query) AnnotationExists(annotation string) *Query {
	q.annotations[annotation] = ""
	return q
}

func (q *Query) AnnotationIs(annotation string, value string) *Query {
	q.annotations[annotation] = value
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
