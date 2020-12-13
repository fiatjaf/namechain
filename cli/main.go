package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/docopt/docopt-go"
	"github.com/fiatjaf/namechain/common"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
)

var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
var config common.Config

const USAGE = `namecli

Usage:
  namecli <method> <params>...
`

func main() {
	// find datadir
	flag.StringVar(&config.DataDir, "datadir", "~/.namechain", "the base directory we will use to read your config file from and store data into.")
	flag.Parse()
	config.DataDir, _ = homedir.Expand(config.DataDir)

	// read config file
	config.ReadConfig()

	// parse args
	opts, err := docopt.ParseDoc(USAGE)
	if err != nil {
		return
	}

	// run the RPC call
	method := opts["<method>"].(string)
	params := opts["<params>"].([]string)

	jreq, _ := json.Marshal(common.RPCRequest{
		Method: method,
		Params: params,
	})
	r, err := http.Post(config.RPCAddr, "application/json", bytes.NewReader(jreq))
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't reach the rpc server. is named running?")
	}

	defer r.Body.Close()
	body, _ := ioutil.ReadAll(r.Body)
	var resp common.RPCResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		log.Fatal().Err(err).Str("body", string(body)).
			Msg("got an invalid response from rpc server")
	}

	var printable []byte
	if resp.Error.Code != 0 {
		printable, _ = json.Marshal(resp.Error)
	} else {
		printable, _ = json.Marshal(resp.Result)
	}

	fmt.Println(string(printable))
}
