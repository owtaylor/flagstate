package main

import (
	"github.com/docker/distribution/digest"
)

type Image struct {
	Digest       digest.Digest
	MediaType    string
	OS           string
	Architecture string
	Annotations  map[string]string `json:",omitempty"`
	Labels       map[string]string `json:",omitempty"`
}

type TaggedImage struct {
	Image
	Tags []string
}

type ImageList struct {
	Digest      digest.Digest
	MediaType   string
	Images      []*Image
	Annotations map[string]string `json:",omitempty"`
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
	} else if v := im.Labels["org.label-schema.name"]; v != "" {
		return v
	} else if v := im.Labels["io.k8s.display-name"]; v != "" {
		return v
	} else if v := im.Labels["name"]; v != "" {
		return v
	} else if v := im.Labels["Name"]; v != "" {
		return v
	} else {
		return ""
	}
}
func (im *Image) Description() string {
	if v := im.Annotations["org.opencontainers.image.description"]; v != "" {
		return v
	} else if v := im.Labels["org.label-schema.description"]; v != "" {
		return v
	} else if v := im.Labels["io.k8s.description"]; v != "" {
		return v
	} else if v := im.Labels["description"]; v != "" {
		return v
	} else if v := im.Labels["Description"]; v != "" {
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
