package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger"
	"github.com/fiatjaf/namechain/common"
	"github.com/kr/pretty"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	rpcclient "github.com/stevenroose/go-bitcoin-core-rpc"
)

var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
var config *common.Config
var kvdb *badger.DB
var blocksdb *badger.DB
var chainstatedb *badger.DB
var bitcoin *rpcclient.Client

var (
	DB_KV         = "kv.db"
	DB_BLOCKS     = "blocks.db"
	DB_CHAINSTATE = "chainstate.db"
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

	// initiate databases
	dbpath := filepath.Join(config.DataDir, DB_KV)
	kvdb, err = badger.Open(badger.DefaultOptions(dbpath))
	if err != nil {
		log.Fatal().Err(err).Str("path", dbpath).Msg("failed to open database")
	}
	dbpath = filepath.Join(config.DataDir, DB_BLOCKS)
	blocksdb, err = badger.Open(badger.DefaultOptions(dbpath))
	if err != nil {
		log.Fatal().Err(err).Str("path", dbpath).Msg("failed to open database")
	}
	dbpath = filepath.Join(config.DataDir, DB_CHAINSTATE)
	chainstatedb, err = badger.Open(badger.DefaultOptions(dbpath))
	if err != nil {
		log.Fatal().Err(err).Str("path", dbpath).Msg("failed to open database")
	}

	// load chainstate to memory because why not
	if err := loadChainState(); err != nil {
		log.Fatal().Err(err).Msg("failed to load chainstate")
	}

	// initiate bitcoind connection
	bitcoin = common.OpenBitcoinRPC(config.BitcoinRPC)
	_, err = bitcoin.GetBlockChainInfo()
	if err != nil {
		log.Fatal().Err(err).Interface("params", config.BitcoinRPC).
			Msg("failed to connect to bitcoind RPC")
	}

	// monitor the bitcoin chain
	// this will also give us all the spacechain blocks
	go watchBitcoinBlocks()

	// listen for rpc commands
	// this will also block here
	listenRPC()
}
