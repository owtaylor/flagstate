package flagstate

import (
	"github.com/go-yaml/yaml"
	"io/ioutil"
	"time"
)

type Duration struct {
	Value time.Duration
}

func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	d.Value = parsed
	return nil
}

type Config struct {
	Registry struct {
		Url       string
		PublicUrl string `yaml:"public_url"`
	}
	Components struct {
		WebUI          bool `yaml:"web_ui"`
		AssertEndpoint bool `yaml:"assert_endpoint"`
	}
	Events struct {
		Token string
	}
	Database struct {
		Postgres struct {
			Url string
		}
	}
	Cache struct {
		MaxAgeIndex Duration `yaml:"max_age_index"`
		MaxAgeHtml  Duration `yaml:"max_age_html"`
	}
	Interval struct {
		FetchAll       Duration `yaml:"fetch_all"`
		GarbageCollect Duration `yaml:"garbage_collect"`
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
