package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/kelseyhightower/envconfig"
)

const (
	modeDevelopment = "development"
	modeProduction  = "production"
)

// Config holds the configuration data
type Config struct {
	AppEnv   string `json:"app_env" envconfig:"APP_ENV" default:"development"`
	Address  string `json:"address" envconfig:"ADDRESS" default:"localhost:3000"`
	Managers string `json:"managers" envconfig:"MANAGERS" default:"127.0.0.1:8085"`
	//	DefaultBatchSize           int64  `json:"default_batch_size" envconfig:"DEFAULT_BATCH_SIZE" default:"0"`
	//	DefaultHeightRangeInterval int64  `json:"default_height_range_interval" envconfig:"DEFAULT_HEIGHT_RANGE_INTERVAL" default:"0"`
	CodaEndpoint string `json:"coda_endpoint" envconfig:"CODA_ENDPOINT" required:"true"`
}

// FromFile reads the config from a file
func FromFile(path string, config *Config) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, config)
}

// FromEnv reads the config from environment variables
func FromEnv(config *Config) error {
	return envconfig.Process("", config)
}
