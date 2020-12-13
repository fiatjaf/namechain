package main

import (
	"flag"
	"net/url"
	"os"
	"path/filepath"

	"github.com/fiatjaf/namechain/common"
	"github.com/kr/pretty"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	rpcclient "github.com/stevenroose/go-bitcoin-core-rpc"
	"go.etcd.io/bbolt"
)

var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
var config *common.Config
var db *bbolt.DB
var bitcoin *rpcclient.Client

var (
	BUCKET_KV         = []byte("kv")
	BUCKET_BLOCKS     = []byte("blocks")
	BUCKET_CHAINSTATE = []byte("chainstate")
)

func main() {
	var err error
	config = &common.Config{}

	// find datadir
	flag.StringVar(&config.DataDir, "datadir", "~/.namechain", "the base directory we will use to read your config file from and store data into.")
	flag.Parse()
	config.DataDir, _ = homedir.Expand(config.DataDir)

	// read config file
	config.ReadConfig()
	pretty.Log(config)

	// initiate database
	dbpath := filepath.Join(config.DataDir, "db.bolt")
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

	// monitor the bitcoin chain
	// this will also give us all the spacechain blocks
	go watchBitcoinBlocks()

	// listen for rpc commands
	// this will also block here
	listenRPC()
}
