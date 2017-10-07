package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/ocischema"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/opencontainers/image-spec/specs-go/v1"
	"io"
	"log"
	"sort"
)

type Fetcher struct {
	db          Database
	changes     *ChangeBroadcaster
	registryUrl string
	channel     chan fetchRequest
}

type requestType int

const (
	requestNone = iota
	requestFetchAll
	requestFetchRepository
	requestGarbageCollect
)

type fetchRequest struct {
	which       requestType
	repository  string
	lowPriority bool
}

func NewFetcher(db Database, changes *ChangeBroadcaster, registryUrl string) *Fetcher {
	f := Fetcher{
		db:          db,
		changes:     changes,
		registryUrl: registryUrl,
		channel:     make(chan fetchRequest, 100),
	}

	go f.dispatch()

	return &f
}

func (f *Fetcher) FetchAll() {
	f.channel <- fetchRequest{
		which: requestFetchAll,
	}
}

func (f *Fetcher) FetchRepository(repository string) {
	f.channel <- fetchRequest{
		which:      requestFetchRepository,
		repository: repository,
	}
}

func (f *Fetcher) GarbageCollect() {
	f.channel <- fetchRequest{
		which: requestGarbageCollect,
	}
}

func (f *Fetcher) dispatch() {
	ctx := context.Background()

	// Start a pool of goroutines that will fetch information about
	// repositories.
	dispatcher := NewRepoDispatcher()
	for i := 0; i < 5; i++ {
		go func() {
			ctx := context.Background()
			for true {
				repo := dispatcher.Take()
				err := f.fetchRepository(ctx, repo)
				if err != nil {
					log.Printf("Error fetching %s: %v", repo, err)
				}
				dispatcher.Release(repo)
			}
		}()
	}

	for true {
		request := <-f.channel
		switch request.which {
		case requestFetchAll:
			dispatcher.Lock()
			err := f.fetchAll(ctx)
			if err != nil {
				log.Printf("Error fetching all repositories: %v", err)
			}
			dispatcher.Unlock()
		case requestFetchRepository:
			dispatcher.Add(request.repository, request.lowPriority)
		case requestGarbageCollect:
			dispatcher.Lock()
			err := f.garbageCollect(ctx)
			dispatcher.Unlock()
			if err != nil {
				log.Printf("Error removing unused images: %v", err)
			}
		}
	}
}

func (f *Fetcher) garbageCollect(ctx context.Context) error {
	tx, err := f.db.Begin(ctx)
	if err != nil {
		return err
	}

	err = tx.DeleteUnused()
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (f *Fetcher) checkModification(tx Tx) {
	modified, _ := tx.Modified()
	if modified {
		f.changes.Change()
	}
}

func (f *Fetcher) fetchAll(ctx context.Context) error {
	const pageSize = 100

	trans := transport.NewTransport(nil)
	registry, err := client.NewRegistry(ctx, f.registryUrl, trans)
	if err != nil {
		return err
	}

	last := ""
	allRepos := make(map[string]bool)
	page := make([]string, pageSize)
	for {
		filled, err := registry.Repositories(ctx, page, last)
		if err != nil && err != io.EOF {
			return err
		}
		if filled == 0 {
			break
		}
		last = page[filled-1]

		for _, repo := range page[:filled] {
			allRepos[repo] = true
		}

		if err == io.EOF {
			break
		}
	}

	tx, err := f.db.Begin(ctx)
	if err != nil {
		return err
	}

	err = tx.DeleteMissingRepos(allRepos)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	f.checkModification(tx)

	for r := range allRepos {
		f.channel <- fetchRequest{
			which:       requestFetchRepository,
			repository:  r,
			lowPriority: true,
		}
	}

	return nil
}

func (f *Fetcher) fetchImage(op *fetchOperation, dgst digest.Digest, image *Image) error {
	mfst, err := op.manifests.Get(op.ctx, dgst)
	if err != nil {
		return err
	}

	image.Digest = dgst
	image.Annotations = make(map[string]string)

	var labelMap *map[string]interface{}

	switch v := mfst.(type) {
	case *schema2.DeserializedManifest:
		image.MediaType = v.MediaType

		bytes, err := op.blobs.Get(op.ctx, v.Config.Digest)
		if err != nil {
			return err
		}
		config := make(map[string]interface{})
		err = json.Unmarshal(bytes, &config)
		if err != nil {
			return err
		}
		architecture, ok := config["architecture"].(string)
		if ok {
			image.Architecture = architecture
		}

		os, ok := config["os"].(string)
		if ok {
			image.OS = os
		}

		configEntry, ok := config["config"].(map[string]interface{})
		if ok {
			labels, ok := configEntry["Labels"].(map[string]interface{})
			if ok {
				labelMap = &labels
			}
		}
	case *ocischema.DeserializedManifest:
		image.MediaType = v.MediaType

		for key, value := range v.Annotations {
			image.Annotations[key] = value
		}

		bytes, err := op.blobs.Get(op.ctx, v.Config.Digest)
		if err != nil {
			return err
		}
		config := make(map[string]interface{})
		err = json.Unmarshal(bytes, &config)
		if err != nil {
			return err
		}
		architecture, ok := config["architecture"].(string)
		if ok {
			image.Architecture = architecture
		}

		os, ok := config["os"].(string)
		if ok {
			image.OS = os
		}

		configEntry, ok := config["config"].(map[string]interface{})
		if ok {
			labels, ok := configEntry["Labels"].(map[string]interface{})
			if ok {
				labelMap = &labels
			}
		}
	default:
		return fmt.Errorf("Can't handle manifest %T", mfst)
	}

	if labelMap != nil {
		for label, value := range *labelMap {
			valueString, ok := value.(string)
			if ok {
				image.Annotations["label:"+label] = valueString
			}
		}
	}

	return nil
}

func (f *Fetcher) fetchImageList(op *fetchOperation, dgst digest.Digest, list *ImageList) error {
	mfst, err := op.manifests.Get(op.ctx, dgst)
	if err != nil {
		return err
	}

	list.Digest = dgst
	list.Annotations = make(map[string]string)

	switch v := mfst.(type) {
	case *manifestlist.DeserializedManifestList:
		list.MediaType = v.MediaType
		for _, descriptor := range v.Manifests {
			var image Image
			err := f.fetchImage(op, descriptor.Digest, &image)
			if err != nil {
				return err
			}
			list.Images = append(list.Images, &image)
		}
	default:
		return fmt.Errorf("Can't handle manifest %T", mfst)
	}

	return nil
}

func (f *Fetcher) getTagsFromRegistry(op *fetchOperation) (imageTags map[string]digest.Digest, listTags map[string]digest.Digest, err error) {
	imageTags = make(map[string]digest.Digest)
	listTags = make(map[string]digest.Digest)

	allTags, err := op.tags.All(op.ctx)
	if err != nil {
		return
	}

	for t, digest := range allTags {
		descriptor, e := op.tags.Get(op.ctx, digest)
		if e != nil {
			err = e
			return
		}

		switch descriptor.MediaType {
		case schema2.MediaTypeManifest:
			imageTags[allTags[t]] = descriptor.Digest
		case manifestlist.MediaTypeManifestList:
			listTags[allTags[t]] = descriptor.Digest
		case v1.MediaTypeImageManifest:
			imageTags[allTags[t]] = descriptor.Digest
		case v1.MediaTypeImageIndex:
			listTags[allTags[t]] = descriptor.Digest
		default:
			continue
		}
	}

	return
}

func stringsEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (f *Fetcher) updateRepositoryInDatabase(op *fetchOperation, tx Tx, imageTags map[string]digest.Digest, listTags map[string]digest.Digest) error {
	repository := op.repo.Named().Name()
	repositories, err := tx.DoQuery(NewQuery().Repository(repository))
	if err != nil {
		return err
	}

	oldImages := make(map[digest.Digest]*TaggedImage)
	oldLists := make(map[digest.Digest]*TaggedImageList)
	if len(repositories) > 0 {
		oldRepo := repositories[0]
		for _, image := range oldRepo.Images {
			oldImages[image.Digest] = image
		}
		for _, list := range oldRepo.Lists {
			oldLists[list.Digest] = list
		}
	}

	changed := false

	newImages := make(map[digest.Digest][]string)
	for tag, dgst := range imageTags {
		newImages[dgst] = append(newImages[dgst], tag)
	}

	for dgst, newTags := range newImages {
		sort.Strings(newTags)
		oldImage := oldImages[dgst]
		if oldImage == nil {
			var image TaggedImage
			err := f.fetchImage(op, dgst, &image.Image)
			if err != nil {
				return err
			}
			image.Tags = newTags
			err = tx.StoreImage(repository, &image)
			if err != nil {
				return err
			}
			changed = true
		} else if !stringsEqual(oldImage.Tags, newTags) {
			err = tx.SetImageTags(repository, dgst, newTags)
			if err != nil {
				return err
			}
			changed = true
		}

		delete(oldImages, dgst)
	}

	for dgst := range oldImages {
		tx.DeleteImage(repository, dgst)
		changed = true
	}

	newLists := make(map[digest.Digest][]string)
	for tag, dgst := range listTags {
		newLists[dgst] = append(newLists[dgst], tag)
	}

	for dgst, newTags := range newLists {
		sort.Strings(newTags)
		oldList := oldLists[dgst]
		if oldList == nil {
			var list TaggedImageList
			err := f.fetchImageList(op, dgst, &list.ImageList)
			if err != nil {
				return err
			}
			list.Tags = newTags
			err = tx.StoreImageList(repository, &list)
			if err != nil {
				return err
			}
			changed = true
		} else if !stringsEqual(oldList.Tags, newTags) {
			err = tx.SetImageListTags(repository, dgst, newTags)
			if err != nil {
				return err
			}
			changed = true
		}

		delete(oldLists, dgst)
	}

	for dgst := range oldLists {
		tx.DeleteImageList(repository, dgst)
		changed = true
	}

	if changed {
		f.changes.Change()
	}

	return nil
}

type fetchOperation struct {
	fetcher *Fetcher
	ctx     context.Context

	repo      distribution.Repository
	blobs     distribution.BlobService
	tags      distribution.TagService
	manifests distribution.ManifestService
}

func (f *Fetcher) newFetchOperation(ctx context.Context, repository string) (*fetchOperation, error) {
	op := &fetchOperation{
		ctx:     ctx,
		fetcher: f,
	}

	trans := transport.NewTransport(nil)

	ref, err := reference.ParseNamed(repository)
	if err != nil {
		return nil, err
	}

	op.repo, err = client.NewRepository(ctx, ref, f.registryUrl, trans)
	if err != nil {
		return nil, err
	}

	op.tags = op.repo.Tags(op.ctx)

	op.manifests, err = op.repo.Manifests(op.ctx)
	if err != nil {
		return nil, err
	}
	op.blobs = op.repo.Blobs(op.ctx)

	return op, nil
}

func (f *Fetcher) fetchRepository(ctx context.Context, repository string) error {
	op, err := f.newFetchOperation(ctx, repository)
	if err != nil {
		return err
	}

	imageTags, listTags, err := f.getTagsFromRegistry(op)
	if err != nil {
		return err
	}

	tx, err := f.db.Begin(ctx)
	if err != nil {
		return err
	}

	err = f.updateRepositoryInDatabase(op, tx, imageTags, listTags)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	f.checkModification(tx)

	return nil
}
