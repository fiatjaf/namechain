package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
)

func downloadBlock(magnetHash []byte) (serializedBlock []byte, err error) {
	// start a torrent client and try to download this block
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
