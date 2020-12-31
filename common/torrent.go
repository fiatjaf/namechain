package common

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"net"
	"path/filepath"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
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

func blockTorrentHash(serializedBlock []byte) (metainfo.Hash, error) {
	mi := metainfo.MetaInfo{
		AnnounceList: make([][]string, 0),
	}

	blockSize := len(serializedBlock)
	blockHash := sha256.Sum256(serializedBlock)

	info := metainfo.Info{
		PieceLength: 256 * 1024,
		Files: []metainfo.FileInfo{
			{Length: int64(blockSize), Path: []string{hex.EncodeToString(blockHash[:])}},
		},
	}

	err := info.GeneratePieces(func(fi metainfo.FileInfo) (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewBuffer(serializedBlock)), nil
	})
	if err != nil {
		return metainfo.Hash{}, err
	}

	mi.InfoBytes, err = bencode.Marshal(info)
	if err != nil {
		return metainfo.Hash{}, err
	}

	infohash := mi.HashInfoBytes()
	return infohash, nil
}
