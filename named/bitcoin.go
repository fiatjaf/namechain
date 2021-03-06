package main

import (
	"bytes"
	"strconv"
	"time"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/dgraph-io/badger"
)

const (
	// in which block and tx the chain was started
	GENESIS_BLOCK = 670000
	GENESIS_TXID  = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	// keys for db
	LAST_SCANNED_BLOCK = "last-scanned-block"
	LAST_SEEN_TXID     = "last-seen-txid"
)

func watchBitcoinBlocks() {
	var lastScannedBlock int
	var lastSpottedTxid *chainhash.Hash

	// load checkpoints
	if err := kvdb.View(func(txn *badger.Txn) error {
		if v, err := txn.Get([]byte(LAST_SCANNED_BLOCK)); err == badger.ErrKeyNotFound {
			lastScannedBlock = GENESIS_BLOCK
			lastSpottedTxid, _ = chainhash.NewHashFromStr(GENESIS_TXID)
		} else if err != nil {
			return err
		} else {
			lastScannedBlock, _ = strconv.Atoi(v.String())

			if v, err := txn.Get([]byte(LAST_SEEN_TXID)); err != nil {
				return err
			} else {
				v.Value(func(val []byte) error {
					lastSpottedTxid.SetBytes(val)
					return nil
				})
			}
		}

		return nil
	}); err != nil {
		log.Fatal().Err(err).Msg("failed to load our bitcoin checkpoints")
	}

	// instantiate variables
	var (
		relevantTxHash    chainhash.Hash
		payingChild       *wire.MsgTx
		spacechainBlockId metainfo.Hash
		serializedBlock   []byte
	)

	// start scanning
	for {
		lastScannedBlock++

		hash, err := bitcoin.GetBlockHash(int64(lastScannedBlock))
		if err != nil {
			log.Info().Int("block", lastScannedBlock).
				Msg("this block doesn't exist yet, let's wait 2 minutes")
			time.Sleep(2 * time.Minute)
			continue
		}

		block, _ := bitcoin.GetBlock(hash)
		var relevantTx *wire.MsgTx
		for _, tx := range block.Transactions {
			for _, inp := range tx.TxIn {
				if inp.PreviousOutPoint.Hash.IsEqual(lastSpottedTxid) {
					// found. it means this tx contains the next spacechain block.
					relevantTx = tx
					goto foundTx
				}
			}
		}

		// we didn't find anything, go to the next block.
		goto saveCheckpoints

	foundTx:
		// this transaction contains a spacechain block
		relevantTxHash = relevantTx.TxHash()
		// let's find the child who is paying for it with CPFP
		// it must be a transaction in this same block (please, miners)
		for _, tx := range block.Transactions {
			for _, inp := range tx.TxIn {
				if inp.PreviousOutPoint.Hash.IsEqual(&relevantTxHash) {
					payingChild = tx
					goto foundChild
				}
			}
		}

		// we didn't find anything. it means miners did it wrong and not include
		// both transactions in the same block.
		log.Fatal().Int("block", lastScannedBlock).Str("tx", relevantTxHash.String()).
			Msg("miners are wrong! please report.")

	foundChild:
		// now we search for an OP_RETURN here which contains the spacechain block id
		for _, out := range payingChild.TxOut {
			if bytes.HasPrefix(out.PkScript, []byte{
				106 /* OP_RETURN */, 14 /* 20 bytes */}) {
				for i, b := range out.PkScript[2:] {
					spacechainBlockId[i] = b
				}
				break
			}
		}

		// later we can implement a queue here to download all blocks
		// concurrently but process sequentially.
		// for now let's just download sequentially too.
		log.Info().Str("id", spacechainBlockId.HexString()).
			Msg("downloading spacechain block")
		serializedBlock = <-downloadBlock(spacechainBlockId)

		err = addBlock(serializedBlock)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to process block")
		}

		// assign this here so we can save it
		lastSpottedTxid = &relevantTxHash

	saveCheckpoints:
		// save checkpoints
		if err := kvdb.Update(func(txn *badger.Txn) error {
			if err := txn.Set(
				[]byte(LAST_SCANNED_BLOCK),
				[]byte(strconv.Itoa(lastScannedBlock)),
			); err != nil {
				return err
			}

			if err := txn.Set(
				[]byte(LAST_SEEN_TXID),
				relevantTxHash[:],
			); err != nil {
				return err
			}

			return nil
		}); err != nil {
			log.Fatal().Err(err).Msg("failed to save checkpoints on db")
		}
	}
}
