package config

import (
	"bytes"
	"errors"
	"github.com/spf13/viper"
	"github.com/xtforgame/agak/requestsender"
	"io/ioutil"
)

type AppConfig struct {
	Image       string `yaml:"image"`
	Container   string `yaml:"container"`
	Endpoint    string `yaml:"endpoint"`
	Network     string `yaml:"network"`
	ExposedPort string `yaml:"exposedPort"`

	EkeyMountPath   string `yaml:"ekeyMountPath"`
	ConfigMountPath string `yaml:"configMountPath"`
	IsSecondary     bool   `yaml:"isSecondary"`
}

type Config struct {
	RequestSender requestsender.RequestSenderConfig `yaml:"requestSender"`
}

func ParseConfig(configFilename string) (*Config, error) {
	if configFilename == "" {
		return nil, errors.New("no config provided")
	}

	viper.SetConfigType("yaml")
	viper.SetConfigName(configFilename)

	// viper.SetDefault("ContentDir", "content")
	// viper.SetDefault("LayoutDir", "layouts")
	// viper.SetDefault("Taxonomies", map[string]string{"tag": "tags", "category": "categories"})
	// viper.SetDefault("TaxonomiesX", []string{"tags", "categories"})

	// viper.WriteConfigAs("./config.yml")
	bs, err := ioutil.ReadFile(configFilename)
	if err != nil {
		return nil, err
	}
	err = viper.ReadConfig(bytes.NewBuffer(bs))
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = viper.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
