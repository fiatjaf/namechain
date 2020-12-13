package common

type Config struct {
	DataDir string

	BitcoinRPC string `yaml:"bitcoin-rpc"` // 'http://user:pass@localhost:18843'
	IPv4       string `yaml:"ipv4"`
	IPv6       string `yaml:"ipv6"`
	ListenAddr string `yaml:"listen-addr"`
}
