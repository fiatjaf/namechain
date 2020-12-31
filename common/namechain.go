package common

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/cbergoon/merkletree"
	"go.uber.org/zap/buffer"
)

type Block struct {
	PreviousBlock metainfo.Hash
	MerkleRoot    []byte
	BlockHash     []byte               // sha256(previousBlock, merkleRoot)
	Transactions  []merkletree.Content // this is []Transaction underneath

	// ID is the torrent infohash of the block that will be published to both torrent
	// trackers and the Bitcoin BMM transaction as an OP_RETURN.
	//
	// It is calculated by the procedure specified at the blockTorrentHash() function.
	ID metainfo.Hash
}

func (block Block) MerkleTree() *merkletree.MerkleTree {
	tree, err := merkletree.NewTree(block.Transactions)
	if err != nil {
		log.Fatal().Err(err).Interface("block", block).
			Msg("failed to make merkle tree")
	}
	return tree
}

func ParseBlock(serializedBlock []byte) (block Block, err error) {
	if len(serializedBlock) < 52 {
		return block, errors.New("serialized block is too short")
	}
	if len(serializedBlock) > 200000 {
		return block, errors.New("serialized block is too large")
	}

	// grab this from serialized format
	block.PreviousBlock = metainfo.HashBytes(serializedBlock[0:20])

	// deserialize transactions
	reader := bufio.NewReader(bytes.NewBuffer(serializedBlock[52:]))
	for {
		n, err := reader.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			return block, fmt.Errorf("error reading transaction length: %w", err)
		}
		// n is the size in bytes of the next transaction
		txbytes := make([]byte, int(n))
		_, err = io.ReadFull(reader, txbytes)
		if err != nil {
			return block, fmt.Errorf("error reading transaction bytes: %w", err)
		}
		tx, err := ParseTransaction(txbytes)
		if err != nil {
			return block, fmt.Errorf("error parsing transaction: %w", err)
		}
		block.Transactions = append(block.Transactions, tx)
	}

	// calculate these:
	block.ID, err = blockTorrentHash(serializedBlock)
	if err != nil {
		return block, err
	}
	block.MerkleRoot = block.MerkleTree().MerkleRoot()

	root := block.MerkleTree().MerkleRoot()
	hash := sha256.New()
	hash.Write(block.PreviousBlock.Bytes())
	hash.Write(root)
	block.BlockHash = hash.Sum(nil)

	// check if values match the serialized values
	if bytes.Compare(block.BlockHash, serializedBlock[20:52]) != 0 {
		return block, errors.New("block hash does not match")
	}

	return block, nil
}

func (block Block) Serialize() []byte {
	buf := buffer.Buffer{}

	// previous block
	previous := block.PreviousBlock.Bytes()
	buf.Write(previous)

	// sha256(previous block, merkle root)
	root := block.MerkleTree().MerkleRoot()
	hash := sha256.New()
	hash.Write(previous)
	hash.Write(root)
	buf.Write(hash.Sum(nil))
	// end of reader

	// the transactions go here now
	for _, txc := range block.Transactions {
		b := txc.(Transaction).Serialize()
		txlen := len(b) // serialized transactions cannot be larger than 255
		buf.Write([]byte{uint8(txlen)})
		buf.Write(b)
	}

	return buf.Bytes()
}

type Transaction struct {
	Type uint8

	// these fields may be set or not depending on the type
	Key         [32]byte
	Name        string
	NameHash    [32]byte
	PublishHash [20]byte
}

const (
	TYPE_ACQUIRE  uint8 = 1
	TYPE_TRANSFER uint8 = 2
	TYPE_RENEW    uint8 = 3
	TYPE_PUBLISH  uint8 = 4
)

func ParseTransaction(serialized []byte) (tx Transaction, err error) {
	tx.Type = serialized[0]

	switch tx.Type {
	case TYPE_ACQUIRE:
		if len(serialized) != 66 {
			return tx, errors.New("invalid transaction size")
		}
		copy(tx.Key[:], serialized[1:33])       // pubkey of the acquirer
		copy(tx.NameHash[:], serialized[34:66]) // sha256(name)
	case TYPE_TRANSFER:
		if len(serialized) != 33 {
			return tx, errors.New("invalid transaction size")
		}
		copy(tx.NameHash[:], serialized[1:33]) // sha256(target_pubkey + name)
	case TYPE_RENEW:
		if len(serialized) != 66 {
			return tx, errors.New("invalid transaction size")
		}
		copy(tx.NameHash[:], serialized[1:33]) // sha256(previous_block_id + name)
	case TYPE_PUBLISH:
		if len(serialized) < 23 || len(serialized) > 255 {
			return tx, errors.New("invalid transaction size")
		}
		copy(tx.PublishHash[:], serialized[1:21])
		tx.Name = string(serialized[22:])
	default:
		return tx, fmt.Errorf("unrecognized transaction type %d", tx.Type)
	}

	return tx, nil
}

func (tx Transaction) Serialize() []byte {
	buf := bytes.Buffer{}

	// type
	buf.Write([]byte{tx.Type})

	switch tx.Type {
	case TYPE_ACQUIRE:
		buf.Write(tx.Key[:])
		buf.Write(tx.NameHash[:])
	case TYPE_TRANSFER:
		buf.Write(tx.NameHash[:])
	case TYPE_RENEW:
		buf.Write(tx.NameHash[:])
	case TYPE_PUBLISH:
		buf.Write(tx.PublishHash[:])
		buf.Write([]byte(tx.Name))
	}

	return buf.Bytes()
}

func (tx Transaction) CalculateHash() ([]byte, error) {
	hash := sha256.Sum256(tx.Serialize())
	return hash[:], nil
}

func (tx Transaction) Equals(other merkletree.Content) (bool, error) {
	return bytes.Compare(tx.Serialize(), other.(Transaction).Serialize()) == 0, nil
}

func processBlock(serializedBlock []byte) error {
	return nil
}
