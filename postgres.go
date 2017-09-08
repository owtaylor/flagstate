package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/docker/distribution/digest"
	_ "github.com/lib/pq"
	"log"
	"sort"
	"strconv"
)

type postgresDatabase struct {
	db *sql.DB
}

type postgresTransaction struct {
	tx *sql.Tx
}

func NewPostgresDB(url string) (Database, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	return &postgresDatabase{
		db: db,
	}, nil
}

func (pdb *postgresDatabase) Begin(ctx context.Context) (Tx, error) {
	ptx := postgresTransaction{}

	var err error
	ptx.tx, err = pdb.db.BeginTx(ctx, nil)
	return ptx, err
}

func (pdb *postgresDatabase) DoQuery(ctx context.Context, query *Query) ([]*Repository, error) {
	tx, err := pdb.Begin(ctx)
	if err != nil {
		return nil, err
	}

	results, err := tx.DoQuery(NewQuery())
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (ptx postgresTransaction) Commit() error {
	return ptx.tx.Commit()
}

func (ptx postgresTransaction) Rollback() error {
	return ptx.tx.Rollback()
}

func makeWhereClause(query *Query) (clause string, args []interface{}) {
	clause = ` WHERE 1 = 1`
	args = make([]interface{}, 0, 20)

	if query.repository != "" {
		args = append(args, query.repository)
		clause += ` AND t.repository = $` + strconv.Itoa(len(args))
	}

	if query.tag != "" {
		args = append(args, query.tag)
		clause += ` AND t.tag = $` + strconv.Itoa(len(args))
	}

	if query.os != "" {
		args = append(args, query.os)
		clause += ` AND i.os = $` + strconv.Itoa(len(args))
	}

	if query.arch != "" {
		args = append(args, query.arch)
		clause += ` AND i.arch = $` + strconv.Itoa(len(args))
	}

	for annotation, value := range query.annotations {
		if value == "" {
			args = append(args, annotation)
			clause += ` AND i.annotations ? $` + strconv.Itoa(len(args))
		} else {
			argJson, _ := json.Marshal(map[string]string{
				annotation: value,
			})
			args = append(args, argJson)
			clause += ` AND i.annotations <@ ` + strconv.Itoa(len(args))
		}
	}

	return
}

func getAnnotations(annotationsJson string) (map[string]string, error) {
	annotations := make(map[string]string)
	var unmarshaled map[string]interface{}
	err := json.Unmarshal([]byte(annotationsJson), &unmarshaled)
	if err != nil {
		return nil, err
	}
	for k, v := range unmarshaled {
		vString, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("Bad annotation value %v: %v", k, v)
		}
		annotations[k] = vString
	}

	return annotations, nil
}

func (ptx postgresTransaction) doImageQuery(query *Query) ([]*Repository, error) {
	whereClause, args := makeWhereClause(query)

	imageQuery := `SELECT i.media_type, i.digest, i.os, i.arch, i.annotations, t.repository, t.tag FROM image i ` +
		`JOIN image_tag t on t.image = i.digest ` +
		whereClause +
		` ORDER BY (t.repository, i.digest)`

	rows, err := ptx.tx.Query(imageQuery, args...)
	if err != nil {
		return make([]*Repository, 0), err
	}

	var result []*Repository = make([]*Repository, 0)
	var currentRepository *Repository
	var currentImage *TaggedImage
	for rows.Next() {
		var mediaType string
		var imageDigest digest.Digest
		var os string
		var arch string
		var imageAnnotationsJson string
		var repository string
		var tag string

		err = rows.Scan(&mediaType, &imageDigest, &os, &arch, &imageAnnotationsJson, &repository, &tag)
		if err != nil {
			return nil, err
		}
		if currentRepository == nil || repository != currentRepository.Name {
			currentRepository = &Repository{
				Name:   repository,
				Images: make([]*TaggedImage, 0),
				Lists:  make([]*TaggedImageList, 0),
			}
			result = append(result, currentRepository)
			currentImage = nil
		}
		if currentImage == nil || imageDigest != currentImage.Digest {
			imageAnnotations, err := getAnnotations(imageAnnotationsJson)
			if err != nil {
				log.Print(err)
				continue
			}

			currentImage = &TaggedImage{
				Image: Image{
					Digest:      imageDigest,
					MediaType:   mediaType,
					OS:          os,
					Arch:        arch,
					Annotations: imageAnnotations,
				},
				Tags: make([]string, 0),
			}
			currentRepository.Images = append(currentRepository.Images, currentImage)
		}

		currentImage.Tags = append(currentImage.Tags, tag)
	}

	return result, nil
}

func (ptx postgresTransaction) doListQuery(query *Query) ([]*Repository, error) {
	whereClause, args := makeWhereClause(query)

	listQuery := `SELECT i.media_type, i.digest, i.os, i.arch, i.annotations, t.repository, t.tag, l.digest, l.annotations FROM image i ` +
		`JOIN list_entry e on e.image = i.digest ` +
		`JOIN list_tag t on t.list = e.list ` +
		`JOIN list l on e.list = l.digest ` +
		whereClause +
		` ORDER BY (t.repository, l.digest, i.digest)`

	rows, err := ptx.tx.Query(listQuery, args...)
	if err != nil {
		return make([]*Repository, 0), err
	}

	var result []*Repository = make([]*Repository, 0)
	var currentRepository *Repository
	var currentList *TaggedImageList
	var currentImage *Image
	for rows.Next() {
		var mediaType string
		var imageDigest digest.Digest
		var os string
		var arch string
		var imageAnnotationsJson string
		var repository string
		var tag string
		var listDigest digest.Digest
		var listAnnotationsJson string

		err = rows.Scan(&mediaType, &imageDigest, &os, &arch, &imageAnnotationsJson, &repository, &tag, &listDigest, &listAnnotationsJson)
		if err != nil {
			return make([]*Repository, 0), err
		}

		if currentRepository == nil || repository != currentRepository.Name {
			currentRepository = &Repository{
				Name:   repository,
				Images: make([]*TaggedImage, 0),
				Lists:  make([]*TaggedImageList, 0),
			}
			result = append(result, currentRepository)
			currentList = nil
			currentImage = nil
		}
		if currentList == nil || listDigest != currentList.Digest {
			listAnnotations, err := getAnnotations(listAnnotationsJson)
			if err != nil {
				log.Print(err)
				continue
			}

			currentList = &TaggedImageList{
				ImageList: ImageList{
					Digest:      listDigest,
					Annotations: listAnnotations,
				},
				Tags: make([]string, 0),
			}
			currentRepository.Lists = append(currentRepository.Lists, currentList)
			currentImage = nil
		}
		if currentImage == nil || imageDigest != currentImage.Digest {
			imageAnnotations, err := getAnnotations(imageAnnotationsJson)
			if err != nil {
				log.Print(err)
				continue
			}

			currentImage = &Image{
				Digest:      imageDigest,
				MediaType:   mediaType,
				OS:          os,
				Arch:        arch,
				Annotations: imageAnnotations,
			}
			currentList.Images = append(currentList.Images, currentImage)
		}
		if len(currentList.Images) == 1 {
			currentList.Tags = append(currentList.Tags, tag)
		}
	}

	return result, nil
}

func (ptx postgresTransaction) DoQuery(query *Query) ([]*Repository, error) {
	imageRepos, err := ptx.doImageQuery(query)
	if err != nil {
		return nil, err
	}
	listRepos, err := ptx.doListQuery(query)
	if err != nil {
		return nil, err
	}

	i := 0
	j := 0
	result := make([]*Repository, 0)
	for i < len(imageRepos) || j < len(listRepos) {
		if i < len(imageRepos) && j < len(listRepos) {
			if imageRepos[i].Name == listRepos[j].Name {
				imageRepos[i].Lists = listRepos[j].Lists
				result = append(result, imageRepos[i])
				i++
				j++
			} else if imageRepos[i].Name < listRepos[j].Name {
				result = append(result, imageRepos[i])
				i++
			} else {
				result = append(result, listRepos[j])
				j++
			}
		} else if i < len(imageRepos) {
			result = append(result, imageRepos[i])
			i++
		} else {
			result = append(result, listRepos[j])
			j++
		}
	}

	for _, repo := range result {
		for _, image := range repo.Images {
			sort.Strings(image.Tags)
		}
		for _, list := range repo.Lists {
			sort.Strings(list.Tags)
		}
	}

	return result, nil
}

func (ptx postgresTransaction) getTags(repository string, target string, dgst digest.Digest) (map[string]bool, error) {
	rows, err := ptx.tx.Query(
		`SELECT tag FROM `+target+`_tag WHERE `+target+` = $1 `,
		dgst)
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for rows.Next() {
		var tag string
		err := rows.Scan(&tag)
		if err != nil {
			return nil, err
		}
		result[tag] = true
	}

	return result, nil
}

func (ptx postgresTransaction) setTags(repository string, target string, dgst digest.Digest, tags []string) error {
	log.Printf("Setting tags for %s %s/%s: %s", target, repository, dgst, tags)
	oldTags, err := ptx.getTags(repository, target, dgst)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		delete(oldTags, tag)
		_, err := ptx.tx.Exec(
			`INSERT INTO `+target+`_tag (repository, tag, `+target+` ) `+
				`VALUES ($1, $2, $3) `+
				`ON CONFLICT (repository, tag) DO UPDATE SET `+target+` = $3 `,
			repository, tag, dgst)

		if err != nil {
			return err
		}
	}

	for tag := range oldTags {
		_, err := ptx.tx.Exec(
			`DELETE FROM `+target+`_tag `+
				`WHERE repository = $1 AND tag = $2 AND `+target+` = $3 `,
			repository, tag, dgst)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ptx postgresTransaction) SetImageTags(repository string, dgst digest.Digest, tags []string) error {
	return ptx.setTags(repository, "image", dgst, tags)
}

func (ptx postgresTransaction) SetImageListTags(repository string, dgst digest.Digest, tags []string) error {
	return ptx.setTags(repository, "list", dgst, tags)
}

func (ptx postgresTransaction) storeImage(repository string, image *Image) error {
	log.Printf("Storing image %s/%s", repository, image.Digest)
	annotationsJson, _ := json.Marshal(image.Annotations)
	_, err := ptx.tx.Exec(
		`INSERT INTO image (digest, media_type, arch, os, annotations) `+
			`VALUES ($1, $2, $3, $4, $5) ON CONFLICT (digest) DO NOTHING `,
		image.Digest, image.MediaType, image.Arch, image.OS, annotationsJson)
	return err
}

func (ptx postgresTransaction) StoreImage(repository string, image *TaggedImage) error {
	err := ptx.storeImage(repository, &image.Image)
	if err != nil {
		return err
	}

	return ptx.SetImageTags(repository, image.Digest, image.Tags)
}

func (ptx postgresTransaction) storeImageList(repository string, list *ImageList) error {
	log.Printf("Storing list %s/%s", repository, list.Digest)
	annotationsJson, _ := json.Marshal(list.Annotations)
	res, err := ptx.tx.Exec(
		`INSERT INTO list (digest, annotations) `+
			`VALUES ($1, $2) ON CONFLICT (digest) DO NOTHING `,
		list.Digest, annotationsJson)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return nil
	}

	for _, image := range list.Images {
		err = ptx.storeImage(repository, image)
		if err != nil {
			return err
		}

		_, err := ptx.tx.Exec(
			`INSERT INTO list_entry (list, image) `+
				`VALUES ($1, $2) ON CONFLICT (list, image) DO NOTHING `,
			list.Digest, image.Digest)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ptx postgresTransaction) StoreImageList(repository string, list *TaggedImageList) error {
	err := ptx.storeImageList(repository, &list.ImageList)
	if err != nil {
		return err
	}

	return ptx.SetImageListTags(repository, list.Digest, list.Tags)
}

func (ptx postgresTransaction) DeleteImage(repository string, dgst digest.Digest) error {
	log.Printf("Deleting tags for image %s/%s", repository, dgst)
	_, err := ptx.tx.Exec(
		`DELETE FROM image_tag WHERE repository = $1 AND image = $2 `,
		repository, dgst)

	return err
}

func (ptx postgresTransaction) DeleteImageList(repository string, dgst digest.Digest) error {
	log.Printf("Deleting tags for image_list %s/%s", repository, dgst)
	_, err := ptx.tx.Exec(
		`DELETE FROM list_tag WHERE repository = $1 AND list = $2 `,
		repository, dgst)

	return err
}

func (ptx postgresTransaction) deleteMissingReposFromTable(table string, allRepos map[string]bool) error {
	toDelete := make([]string, 0)

	rows, err := ptx.tx.Query(`SELECT DISTINCT repository FROM ` + table)
	if err != nil {
		return err
	}

	for rows.Next() {
		var repository string
		err := rows.Scan(&repository)
		if err != nil {
			return err
		}
		if !allRepos[repository] {
			toDelete = append(toDelete, repository)
		}
	}

	for _, repo := range toDelete {
		_, err := ptx.tx.Exec(`DELETE FROM `+table+`_tag WHERE repository = $1`,
			repo)
		if err != nil {
			return err
		}
	}

	return err
}

func (ptx postgresTransaction) DeleteMissingRepos(allRepos map[string]bool) error {
	err := ptx.deleteMissingReposFromTable("image_tag", allRepos)
	if err != nil {
		return err
	}
	err = ptx.deleteMissingReposFromTable("list_tag", allRepos)
	if err != nil {
		return err
	}

	return nil
}

func (ptx postgresTransaction) DeleteUnused() error {
	_, err := ptx.tx.Exec(
		`DELETE FROM list ` +
			`WHERE NOT EXISTS (SELECT * FROM list_tag WHERE list_tag.list = list.digest)`)
	if err != nil {
		return err
	}

	_, err = ptx.tx.Exec(
		`DELETE FROM image ` +
			`WHERE NOT EXISTS (SELECT * FROM image_tag WHERE image_tag.image = image.digest) ` +
			`AND NOT EXISTS (SELECT * FROM list_entry WHERE list_entry.image = image.digest)`)
	if err != nil {
		return err
	}

	return nil
}
