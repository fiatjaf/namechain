package common

import (
	"net/url"

	rpcclient "github.com/stevenroose/go-bitcoin-core-rpc"
)

func OpenBitcoinRPC(uri string) *rpcclient.Client {
	// initiate bitcoind connection
	btcParams, _ := url.Parse(uri)
	password, _ := btcParams.User.Password()
	bitcoin, _ := rpcclient.New(&rpcclient.ConnConfig{
		Host: btcParams.Host,
		User: btcParams.User.Username(),
		Pass: password,
	})
	return bitcoin
}
