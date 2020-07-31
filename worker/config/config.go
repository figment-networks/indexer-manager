package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	"github.com/kelseyhightower/envconfig"
)

const (
	modeDevelopment = "development"
	modeProduction  = "production"
)

var (
	errDatabaseRequired   = errors.New("database credentials are required")
	errRPCAddrRequired    = errors.New("datahub tendermint rpc address is required")
	errDatahubKeyRequired = errors.New("datahub api key is required")
)

// Config holds the configuration data
type Config struct {
	AppEnv                     string `json:"app_env" envconfig:"APP_ENV" default:"development"`
	DefaultBatchSize           int64  `json:"default_batch_size" envconfig:"DEFAULT_BATCH_SIZE" default:"0"`
	DefaultHeightRangeInterval int64  `json:"default_height_range_interval" envconfig:"DEFAULT_HEIGHT_RANGE_INTERVAL" default:"0"`
	TendermintRPCAddr          string `json:"tendermint_rpc_addr" envconfig:"TENDERMINT_RPC_ADDR"`
	DatahubKey                 string `json:"datahub_key" envconfig:"DATAHUB_KEY"`
	FirstBlockHeight           int64  `json:"first_block_height" envconfig:"FIRST_BLOCK_HEIGHT" default:"1"`
	DatabaseURL                string `json:"database_url" envconfig:"DATABASE_URL"`
}

// Validate returns an error if config is invalid
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return errDatabaseRequired
	}

	if c.TendermintRPCAddr == "" {
		return errRPCAddrRequired
	}

	if c.DatahubKey == "" {
		return errDatahubKeyRequired
	}

	return nil
}

// IsDevelopment returns true if app is in dev mode
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == modeDevelopment
}

// IsProduction returns true if app is in production mode
func (c *Config) IsProduction() bool {
	return c.AppEnv == modeProduction
}

// New returns a new config
func New() *Config {
	return &Config{}
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
