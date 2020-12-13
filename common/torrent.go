package common

import (
	"net"
	"path/filepath"

	"github.com/anacrolix/torrent"
)

func TorrentClient(config *Config) (*torrent.Client, error) {
	clientConfig := torrent.NewDefaultClientConfig()
	clientConfig.Seed = true
	clientConfig.DataDir = filepath.Join(config.DataDir, "blocks")
	if config.IPv4 != "" {
		clientConfig.PublicIp4 = net.ParseIP(config.IPv4)
	}
	if config.IPv6 != "" {
		clientConfig.PublicIp6 = net.ParseIP(config.IPv6)
	}
	if config.ListenAddr != "" {
		clientConfig.SetListenAddr(config.ListenAddr)
	}

	client, err := torrent.NewClient(clientConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}
