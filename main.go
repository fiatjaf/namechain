package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/rs/zerolog"
)

var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})

func main() {
	x, err := blockTorrentHash([]byte("abcdefghijklmnopq"))
	if err != nil {
		log.Fatal().Err(err).Msg("error getting block torrent hash")
	}

	fmt.Println(x)
}

func blockTorrentHash(serializedBlock []byte) (string, error) {
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
		return "", err
	}

	mi.InfoBytes, err = bencode.Marshal(info)
	if err != nil {
		return "", err
	}

	infohash := mi.HashInfoBytes()
	return infohash.HexString(), nil
}
