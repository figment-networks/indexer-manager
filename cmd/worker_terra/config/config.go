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
	AppEnv string `json:"app_env" envconfig:"APP_ENV" default:"development"`

	TerraRPCAddr string `json:"terra_rpc_addr" envconfig:"TERRA_RPC_ADDR" required:"true"`
	Address      string `json:"address" envconfig:"ADDRESS" default:"0.0.0.0"`
	Port         string `json:"port" envconfig:"PORT" default:"3000"`

	Managers string `json:"managers" envconfig:"MANAGERS" default:"127.0.0.1:8085"`
	Hostname string `json:"hostname" envconfig:"HOSTNAME"`
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
