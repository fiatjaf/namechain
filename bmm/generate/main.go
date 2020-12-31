package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/psbt"
	"github.com/fiatjaf/namechain/common"
	"github.com/mitchellh/go-homedir"
)

const MIN_OUTPUT_VALUE int64 = 294

var config *common.Config

func main() {
	var err error
	config = &common.Config{}

	var params struct {
		input           string
		numtransactions int64
		blockinterval   int
		genesisfee      int64
		change          string
		disposablekey   string
	}

	flag.StringVar(&config.DataDir, "datadir", "~/.namechain", "the base directory we will use to read your config file from and store data into.")
	flag.StringVar(&params.input, "input", "", "the input vout, in <txid>:<outputnum>")
	flag.Int64Var(&params.numtransactions, "numtransactions", 1,
		"total amount of transactions we will generate")
	flag.IntVar(&params.blockinterval, "blockinterval", 1,
		"relative locktime between bmm transactions")
	flag.Int64Var(&params.genesisfee, "genesisfee", 5000,
		"how much we will pay, in total "+
			"satoshis, for the genesis transaction (on bitcoin)")
	flag.StringVar(&params.change, "change", "", "the change address")
	flag.StringVar(&params.disposablekey, "disposablekey",
		"0000000000000000000000000000000000000000000000000000000000000000",
		"the key we will use to generate the sequence of transactions and discard")
	flag.Parse()

	// find datadir
	config.DataDir, _ = homedir.Expand(config.DataDir)

	// read config file
	config.ReadConfig()

	// use a fixed key if not given
	key, err := hex.DecodeString(params.disposablekey)
	if err != nil {
		log.Fatal(err)
	}
	sk, pk := btcec.PrivKeyFromBytes(btcec.S256(), key)

	// base chain
	var chainParams *chaincfg.Params
	switch {
	case strings.HasPrefix(params.change, "3"), strings.HasPrefix(params.change, "1"),
		strings.HasPrefix(strings.ToLower(params.change), "bc1"):
		chainParams = &chaincfg.MainNetParams
	case strings.HasPrefix(params.change, "2"),
		strings.HasPrefix(params.change, "m"), strings.HasPrefix(params.change, "n"),
		strings.HasPrefix(strings.ToLower(params.change), "tb1"):
		chainParams = &chaincfg.TestNet3Params
	case strings.HasPrefix(strings.ToLower(params.change), "bcrt"):
		chainParams = &chaincfg.SimNetParams
		// case simnnet?
		// chainParams = &chaincfg.RegressionNetParams
	default:
		log.Fatal("invalid chain")
		return
	}

	// genesis transaction input
	spl := strings.Split(params.input, ":")
	inputTxid, _ := chainhash.NewHashFromStr(spl[0])
	outputNum, _ := strconv.Atoi(spl[1])

	// get input amount
	inputTx, err := common.OpenBitcoinRPC(config.BitcoinRPC).
		GetRawTransactionVerbose(inputTxid)
	if err != nil {
		log.Fatal(err)
		return
	}
	inputAmount := int64(math.Round(inputTx.Vout[outputNum].Value * 100000000))

	// change address
	changeAddress, _ := btcutil.DecodeAddress(params.change, chainParams)
	changePkScript, _ := txscript.PayToAddrScript(changeAddress)

	// bmm address and pkscript
	bmmPubKeyHash := btcutil.Hash160(pk.SerializeCompressed())
	bmmAddr, _ := btcutil.NewAddressWitnessPubKeyHash(bmmPubKeyHash, chainParams)
	bmmPkScript, _ := txscript.PayToAddrScript(bmmAddr)

	// the output that will be used by the miner to hook his transaction into
	opTrueScript, _ := txscript.NewScriptBuilder().AddOp(txscript.OP_TRUE).Script()

	// the total amount we will deposit to create the chain of transactions
	fundingAmount := MIN_OUTPUT_VALUE*params.numtransactions + MIN_OUTPUT_VALUE

	// create genesis tx
	genesisTx := wire.NewMsgTx(wire.TxVersion)

	// add input
	genesisTx.AddTxIn(
		wire.NewTxIn(
			wire.NewOutPoint(inputTxid, uint32(outputNum)),
			nil,
			nil,
		),
	)

	// add bmm output
	genesisTx.AddTxOut(
		wire.NewTxOut(fundingAmount, bmmPkScript),
	)

	// add change output
	genesisTx.AddTxOut(
		wire.NewTxOut(inputAmount-fundingAmount-params.genesisfee,
			changePkScript),
	)

	fmt.Printf("spending a total of %d sat to generate this BMM chain.\n",
		fundingAmount+params.genesisfee)

	// print unsigned funding tx
	var serializedGenesis bytes.Buffer
	genesisTx.Serialize(&serializedGenesis)

	psbtPacket, _ := psbt.NewFromUnsignedTx(genesisTx)
	psbtBase64, _ := psbtPacket.B64Encode()
	fmt.Printf("funding transaction to sign (PSBT): %s\n", psbtBase64)

	// get signature for funding tx
	line := bufio.NewReader(os.Stdin)
	fmt.Print("paste finalized PSBT: ")
	finalized, _ := line.ReadString('\n')
	p, err := psbt.NewFromRawBytes(strings.NewReader(finalized), true)
	if err != nil {
		log.Fatal("error parsing psbt: " + err.Error())
		return
	}
	genesisTx, err = psbt.Extract(p)
	if err != nil {
		log.Fatal(err)
		return
	}

	// generate a string of x transactions
	prev := genesisTx
	var i int64
	for i = 0; i < params.numtransactions; i++ {
		tx := wire.NewMsgTx(2)

		prevTxId := prev.TxHash()
		tx.AddTxIn(
			&wire.TxIn{
				PreviousOutPoint: wire.OutPoint{prevTxId, uint32(0)},
				SignatureScript:  nil,
				Witness:          nil,
				Sequence:         uint32(params.blockinterval),
			},
		)
		tx.AddTxOut(
			wire.NewTxOut(fundingAmount-MIN_OUTPUT_VALUE*i, bmmPkScript),
		)
		tx.AddTxOut(
			wire.NewTxOut(MIN_OUTPUT_VALUE, opTrueScript),
		)

		// sign
		sigScript, err := txscript.SignatureScript(tx, 0, bmmPkScript,
			txscript.SigHashAll,
			sk, true)
		if err != nil {
			log.Fatal(err)
			return
		}
		tx.TxIn[0].SignatureScript = sigScript

		// serialize and print
		var serializedTx bytes.Buffer
		tx.Serialize(&serializedTx)

		fmt.Printf("BMM %d: %x \n", i+1, serializedTx.Bytes())

		prev = tx
	}

	// print serialized genesis
	var serializedTx bytes.Buffer
	genesisTx.Serialize(&serializedTx)
	fmt.Printf("\ngenesis: %x \n", serializedTx.Bytes())
	fmt.Printf("publish? [yes/no]: ")
	shouldPublish, _ := line.ReadString('\n')
	if shouldPublish == "yes" {
		_, err := common.OpenBitcoinRPC(config.BitcoinRPC).
			SendRawTransaction(genesisTx, false)
		if err != nil {
			log.Fatal(err)
			return
		}
	}
}
