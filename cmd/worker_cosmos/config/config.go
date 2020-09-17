package config

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/kelseyhightower/envconfig"
)

const (
	modeDevelopment = "development"
	modeProduction  = "production"
)

// Config holds the configuration data
type Config struct {
	AppEnv   string `json:"app_env" envconfig:"APP_ENV" default:"development"`
	Address  string `json:"address" envconfig:"ADDRESS" default:"0.0.0.0"`
	Port     string `json:"port" envconfig:"PORT" default:"3000"`
	HTTPPort string `json:"http_port" envconfig:"HTTP_PORT" default:"8087"`

	TendermintRPCAddr string `json:"tendermint_rpc_addr" envconfig:"TENDERMINT_RPC_ADDR" required:"true"`
	DatahubKey        string `json:"datahub_key" envconfig:"DATAHUB_KEY"`
	ChainID           string `json:"chain_id" envconfig:"CHAIN_ID"`

	Managers        string        `json:"managers" envconfig:"MANAGERS" default:"127.0.0.1:8085"`
	ManagerInterval time.Duration `json:"manager_interval" envconfig:"MANAGER_INTERVAL" default:"10s"`
	Hostname        string        `json:"hostname" envconfig:"HOSTNAME"`

	MaximumHeightsToGet float64 `json:"maximum_heights_to_get" envconfig:"MAXIMUM_HEIGHTS_TO_GET" default:"10000"`
	BigPage             float64 `json:"big_page" envconfig:"BIG_PAGE" default:"1000"`
	RequestsPerSecond   int64   `json:"requests_per_second" envconfig:"REQUESTS_PER_SECOND" default:"33"`
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
