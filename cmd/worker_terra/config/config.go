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
	AppEnv  string `json:"app_env" envconfig:"APP_ENV" default:"development"`
	Address string `json:"address" envconfig:"ADDRESS" default:"localhost:3000"`
	//	DefaultBatchSize           int64  `json:"default_batch_size" envconfig:"DEFAULT_BATCH_SIZE" default:"0"`
	//	DefaultHeightRangeInterval int64  `json:"default_height_range_interval" envconfig:"DEFAULT_HEIGHT_RANGE_INTERVAL" default:"0"`
	TerraRPCAddr     string `json:"terra_rpc_addr" envconfig:"TERRA_RPC_ADDR" required:"true"`
	DatahubKey       string `json:"datahub_key" envconfig:"DATAHUB_KEY" required:"true"`
	FirstBlockHeight int64  `json:"first_block_height" envconfig:"FIRST_BLOCK_HEIGHT" default:"1"`
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
