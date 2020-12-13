package main

import (
	"flag"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/kr/pretty"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	rpcclient "github.com/stevenroose/go-bitcoin-core-rpc"
	"go.etcd.io/bbolt"
	"gopkg.in/yaml.v2"
)

type Config struct {
	datadir string

	BitcoinRPC string `yaml:"bitcoinrpc"` // 'http://user:pass@localhost:18843'
}

var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
var config Config
var db *bbolt.DB
var bitcoin *rpcclient.Client

var (
	BUCKET_KV         = []byte("kv")
	BUCKET_BLOCKS     = []byte("blocks")
	BUCKET_CHAINSTATE = []byte("chainstate")
)

func main() {
	// find datadir
	flag.StringVar(&config.datadir, "datadir", "~/.namechain", "the base directory we will use to read your config file from and store data into.")
	flag.Parse()
	config.datadir, _ = homedir.Expand(config.datadir)

	// read config file
	configFile := filepath.Join(config.datadir, "config.yaml")
	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Info().Err(err).Str("path", configFile).
			Msg("error reading config file, will attempt to create it")
		ioutil.WriteFile(configFile, []byte(""), 0644)
	}
	yaml.Unmarshal(configData, &config)
	pretty.Log(config)

	// initiate database
	dbpath := filepath.Join(config.datadir, "db.bolt")
	db, err = bbolt.Open(dbpath, 0644, nil)
	if err != nil {
		log.Fatal().Err(err).Str("path", dbpath).Msg("failed to open database")
	}

	// initiate bitcoind connection
	btcParams, _ := url.Parse(config.BitcoinRPC)
	password, _ := btcParams.User.Password()
	bitcoin, _ = rpcclient.New(&rpcclient.ConnConfig{
		Host: btcParams.Host,
		User: btcParams.User.Username(),
		Pass: password,
	})
	_, err = bitcoin.GetBlockChainInfo()
	if err != nil {
		log.Fatal().Err(err).Interface("params", btcParams).
			Msg("failed to connect to bitcoind RPC")
	}

	go watchBitcoinBlocks()
}
