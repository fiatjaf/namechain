package common

import (
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Config struct {
	DataDir string

	BitcoinRPC string `yaml:"bitcoin-rpc"` // 'http://user:pass@localhost:18843'
	IPv4       string `yaml:"ipv4"`
	IPv6       string `yaml:"ipv6"`
	ListenAddr string `yaml:"listen-addr"`

	RPCAddr string `yaml:"rpc-addr"`
}

func (c *Config) SetDefaults() {
	if c.RPCAddr == "" {
		c.RPCAddr = "localhost:24335" // 24335 can be read as "named"
	}
}

func (config *Config) ReadConfig() {
	configFile := filepath.Join(config.DataDir, "config.yaml")
	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Info().Err(err).Str("path", configFile).
			Msg("error reading config file, will attempt to create it")
		ioutil.WriteFile(configFile, []byte(""), 0644)
	}
	yaml.Unmarshal(configData, &config)
	config.SetDefaults()
}
