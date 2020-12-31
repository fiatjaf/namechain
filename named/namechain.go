package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/dgraph-io/badger"
	"github.com/fiatjaf/namechain/common"
)

type ChainState struct {
	BlockHeight int
	KnownNames  map[[32]byte]NameData // name hash: data
}

type NameData struct {
	Key              [32]byte
	Name             string
	DataBlobInfoHash [20]byte
}

func loadName(name string) (*NameData, error) {
	nameHash := sha256.Sum256(name)

	var nd NameData
	err := chainstatedb.View(func(txn *badger.Txn) error {
		item := txn.Get(nameHash[:])
		return item.Value(func(v []byte) error {
			var nd NameData
			copy(nd.Key[:], v[0:32])
			copy(nd.DataBlobInfoHash[:], v[33:53])
			nd.Name = string(v[54:])
		})
	})
	return &nd, err
}

func validateTransaction(tx common.Transaction) error {
	// check if operation matches ownership
	switch tx.Type {
	case common.TYPE_ACQUIRE:
		// this name must not have an owner
	case common.TYPE_TRANSFER:
		// signature must be from current owner
	case common.TYPE_RENEW:
		// signature must be from current owner
	// block hash must be from one of the latest 10 blocks
	case common.TYPE_PUBLISH:
		// signature must be from current owner
		// ownership may be of a known name
		//   or of an acquired unknown hash
	}

	// check signature
	// TODO

	return nil
}

func validateBlock(block common.Block) error {
	for i, tx := range block.Transactions {
		if err := validateTransaction(tx.(common.Transaction)); err != nil {
			return fmt.Errorf("error validating transaction %d: %w", i, err)
		}
	}

	return nil
}

func addBlock(serializedBlock []byte) error {
	// parse block
	block, err := common.ParseBlock(serializedBlock)
	if err != nil {
		return fmt.Errorf("error parsing block: %w", err)
	}

	// validate block
	err = validateBlock(block)
	if err != nil {
		return fmt.Errorf("error validating block: %w", err)
	}

	// update and save chainstate
	if err := chainstatedb.Update(func(txn *badger.Txn) error {
		if err := txn.Set(
			[]byte("blockheight"),
			[]byte(strconv.Itoa(chainstate.BlockHeight+1)),
		); err != nil {
			return err
		}

		for _, itx := range block.Transactions {
			tx := itx.(common.Transaction)
			switch tx.Type {
			case TYPE_ACQUIRE:
			case TYPE_TRANSFER:
			case TYPE_RENEW:
			case TYPE_PUBLISH:
			}
		}

		if err := txn.Set(
			[]byte(),
			[]byte(),
		); err != nil {
			return err
		}

		return nil
	}); err != nil {
		log.Fatal().Err(err).Msg("failed to add block")
	}

	// save block
	if err := blocksdb.Update(func(txn *badger.Txn) error {
		if err := txn.Set(
			block.ID[:],
			block.Serialize(),
		); err != nil {
			return err
		}

		buf := make([]byte, 64)
		binary.PutVarint(buf, int64(chainstate.BlockHeight+1))
		if err := txn.Set(
			buf,
			block.ID[:],
		); err != nil {
			return err
		}

		return nil
	}); err != nil {
		log.Fatal().Err(err).Msg("failed to add block")
	}
}

func undoBlock() {}
