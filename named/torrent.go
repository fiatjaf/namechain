package main

import (
	"io/ioutil"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/fiatjaf/namechain/common"
)

func downloadBlock(infohash metainfo.Hash) chan []byte {
	// start a torrent client and try to download this block
	log := log.With().Stringer("block-id", infohash).Logger()

	blockchan := make(chan []byte, 1)
	bt, err := common.TorrentClient(config)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start torrent client")
	}

	blocktorrent, _ := bt.AddTorrentInfoHash(infohash)

	log.Info().Msg("downloading spacechain block")
	go func() {
		psc := blocktorrent.SubscribePieceStateChanges()
		for {
			sc := (<-psc.Values).(torrent.PieceStateChange)
			log.Info().Int("index", sc.Index).Bool("partial", sc.Partial).
				Bool("complete", sc.Complete).Bool("ok", sc.Ok).
				Msg("piece state changed")
		}
	}()
	go func() {
		<-blocktorrent.GotInfo()
		log.Info().Interface("info", blocktorrent.Info()).Msg("got torrent info")
		<-blocktorrent.Closed()
		log.Info().Msg("torrent ended")
	}()

	var downloadComplete chan struct{}
	go func() {
		bt.WaitAll()
		downloadComplete <- struct{}{}
	}()

	go func() {
		defer bt.Close()
		select {
		case <-downloadComplete:
			block, err := ioutil.ReadAll(blocktorrent.NewReader())
			if err != nil {
				log.Fatal().Err(err).Msg("failed to read complete torrent")
			}

			blockchan <- block

			// seed it for 30 minutes more
			log.Info().Msg("seeding")
			time.Sleep(time.Minute * 30)
		case <-time.After(time.Minute * 10):
			log.Fatal().Msg("couldn't download after 10 minutes")
		}
	}()

	log.Info().Msg("block downloaded")
	return blockchan
}
