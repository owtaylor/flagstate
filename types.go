package main

import (
	"github.com/docker/distribution/digest"
)

type Image struct {
	Digest       digest.Digest
	MediaType    string
	OS           string
	Architecture string
	Annotations  map[string]string
}

type TaggedImage struct {
	Image
	Tags []string
}

type ImageList struct {
	Digest      digest.Digest
	MediaType   string
	Images      []*Image
	Annotations map[string]string
}

type TaggedImageList struct {
	ImageList
	Tags []string
}

type Repository struct {
	Name   string
	Images []*TaggedImage
	Lists  []*TaggedImageList
}

func (im *Image) Title() string {
	if v := im.Annotations["org.opencontainers.image.title"]; v != "" {
		return v
	} else if v := im.Annotations["label:org.label-schema.name"]; v != "" {
		return v
	} else if v := im.Annotations["label:io.k8s.display-name"]; v != "" {
		return v
	} else if v := im.Annotations["label:name"]; v != "" {
		return v
	} else if v := im.Annotations["label:Name"]; v != "" {
		return v
	} else {
		return ""
	}
}
func (im *Image) Description() string {
	if v := im.Annotations["org.opencontainers.image.description"]; v != "" {
		return v
	} else if v := im.Annotations["label:org.label-schema.description"]; v != "" {
		return v
	} else if v := im.Annotations["label:io.k8s.description"]; v != "" {
		return v
	} else if v := im.Annotations["label:description"]; v != "" {
		return v
	} else if v := im.Annotations["label:Description"]; v != "" {
		return v
	} else {
		return ""
	}
}

func (im *TaggedImage) IsLatest() bool {
	for _, tag := range im.Tags {
		if tag == "latest" {
			return true
		}
	}

	return false
}
