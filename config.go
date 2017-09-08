package main

import (
	"github.com/go-yaml/yaml"
	"io/ioutil"
)

type Config struct {
	Registry struct {
		Url string
	}
	Events struct {
		Token string
	}
	Database struct {
		Postgres struct {
			Url string
		}
	}
	Interval struct {
		FetchAll       string `yaml:"fetch_all"`
		GarbageCollect string `yaml:"garbage_collect"`
	}
}

func LoadConfig(filename string) (*Config, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
