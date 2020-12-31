package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

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
		bmmindex        int
		spacechainblock string
	}

	flag.StringVar(&config.DataDir, "datadir", "~/.namechain", "the base directory we will use to read your config file from and store data into.")
	flag.IntVar(&params.bmmindex, "bmmindex", 0, "index of the next bmm transaction string")
	flag.StringVar(&params.spacechainblock, "spacechainblock", "", "block id of the spacechain block we're trying to mine")
	flag.Parse()

	// find datadir
	config.DataDir, _ = homedir.Expand(config.DataDir)

	// read config file
	config.ReadConfig()

	blockId, err := hex.DecodeString(params.spacechainblock)
	if err != nil {
		log.Fatal(err)
	}
	bmmIndex, err := strconv.Atoi(params.bmmindex)
	if err != nil {
		log.Fatal(err)
	}

	pregeneratedTxs, err := ioutil.ReadAll("pregenerated")

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
