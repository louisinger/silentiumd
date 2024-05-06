package domain

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"sort"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/sirupsen/logrus"
)

var (
	inputHashTag = []byte("BIP0352/Inputs")
	num_h, _     = hex.DecodeString("50929b74c1a04954b78b4b6035e97a5e078a5a0f28ec96d547bfee9ace803ac0")
)

type TaprootOutput struct {
	Index uint32
	Spent bool
}

type SilentScalar struct {
	TxHash         *chainhash.Hash
	TaprootOutputs []TaprootOutput
	Scalar         []byte

	TxIn []*wire.TxIn
}

func NewSilentScalar(
	tx *btcutil.Tx,
) (*SilentScalar, error) {
	taprootOuts := make([]TaprootOutput, 0, len(tx.MsgTx().TxOut))

	for i, out := range tx.MsgTx().TxOut {
		if txscript.IsPayToTaproot(out.PkScript) {
			taprootOuts = append(taprootOuts, TaprootOutput{
				Index: uint32(i),
				Spent: false,
			})
		}
	}

	if len(taprootOuts) == 0 {
		return nil, ErrNoTaprootOutputs
	}

	return &SilentScalar{
		TxHash:         tx.Hash(),
		TxIn:           tx.MsgTx().TxIn,
		TaprootOutputs: taprootOuts,
	}, nil
}

func (s *SilentScalar) MarkOutputSpent(index uint32) {
	for i, out := range s.TaprootOutputs {
		if out.Index == index {
			s.TaprootOutputs[i].Spent = true
			return
		}
	}
}

func (s *SilentScalar) HasUnspentTaproot() bool {
	for _, out := range s.TaprootOutputs {
		if !out.Spent {
			return true
		}
	}

	return false
}

func (s *SilentScalar) ComputeScalar(
	prevoutGetter func(wire.OutPoint) ([]byte, error),
) error {
	if len(s.Scalar) > 0 {
		return nil
	}

	if len(s.TxIn) == 0 {
		return ErrUnableToComputeScalar
	}

	s.Scalar = computeScalar(s.TxIn, prevoutGetter)
	return nil
}

func computeScalar(
	txIn []*wire.TxIn,
	prevoutGetter func(wire.OutPoint) ([]byte, error),
) []byte {
	publicKeys := getInputPublicKeys(txIn, prevoutGetter)
	sumInputPublicKeys := sumPublicKeys(publicKeys)
	inputHash := getInputHash(txIn, sumInputPublicKeys)
	scalarInputHash := new(big.Int).SetBytes(inputHash.CloneBytes())

	if sumInputPublicKeys == nil {
		// scalar = inputHash * G
		x, y := btcec.S256().ScalarBaseMult(scalarInputHash.Bytes())

		var xFieldVal, yFieldVal btcec.FieldVal
		xFieldVal.SetByteSlice(x.Bytes())
		yFieldVal.SetByteSlice(y.Bytes())

		resultPubKey := btcec.NewPublicKey(&xFieldVal, &yFieldVal)
		return resultPubKey.SerializeCompressed()
	}

	x, y := btcec.S256().ScalarMult(sumInputPublicKeys.X(), sumInputPublicKeys.Y(), inputHash[:])

	var xFieldVal, yFieldVal btcec.FieldVal
	xFieldVal.SetByteSlice(x.Bytes())
	yFieldVal.SetByteSlice(y.Bytes())

	return btcec.NewPublicKey(&xFieldVal, &yFieldVal).SerializeCompressed()
}

func getInputHash(
	txIn []*wire.TxIn,
	sumPublicKeys *btcec.PublicKey,
) *chainhash.Hash {
	outpoints := make([]wire.OutPoint, 0)
	for _, txIn := range txIn {
		outpoints = append(outpoints, txIn.PreviousOutPoint)
	}

	sort.Slice(outpoints, func(i, j int) bool {
		hashComparison := bytes.Compare(outpoints[i].Hash.CloneBytes(), outpoints[j].Hash.CloneBytes())
		if hashComparison != 0 {
			return hashComparison < 0
		}
		return outpoints[i].Index < outpoints[j].Index
	})

	lowestOutpoint := outpoints[0]
	msg := serializeOutpoint(lowestOutpoint)
	if sumPublicKeys != nil {
		msg = append(msg, sumPublicKeys.SerializeCompressed()...)
	}

	return chainhash.TaggedHash(inputHashTag, msg)
}

func serializeOutpoint(outpoint wire.OutPoint) []byte {
	var buf bytes.Buffer
	buf.Write(outpoint.Hash.CloneBytes())

	var index [4]byte
	binary.LittleEndian.PutUint32(index[:], outpoint.Index)
	buf.Write(index[:])

	return buf.Bytes()
}

func sumPublicKeys(publicKeys []*btcec.PublicKey) *btcec.PublicKey {
	if len(publicKeys) == 0 {
		return nil
	}

	if len(publicKeys) == 1 {
		return publicKeys[0]
	}

	x, y := publicKeys[0].X(), publicKeys[0].Y()

	for _, pubkey := range publicKeys[1:] {
		x, y = btcec.S256().Add(x, y, pubkey.X(), pubkey.Y())
	}

	var xFieldVal, yFieldVal btcec.FieldVal
	xFieldVal.SetByteSlice(x.Bytes())
	yFieldVal.SetByteSlice(y.Bytes())

	return btcec.NewPublicKey(&xFieldVal, &yFieldVal)
}

func getInputPublicKeys(
	txIn []*wire.TxIn,
	getPrevoutScript func(wire.OutPoint) ([]byte, error),
) []*btcec.PublicKey {
	publicKeys := make([]*btcec.PublicKey, 0)

	for _, txIn := range txIn {
		pubkey, err := extractPublicKeyFromInput(txIn, getPrevoutScript)
		if err != nil {
			if err != ErrNonStandardScript {
				logrus.Warnf("error extracting public key from input: %v", err)
				continue
			}
		}

		if pubkey != nil {
			publicKeys = append(publicKeys, pubkey)
		}
	}

	return publicKeys
}

func extractPublicKeyFromInput(txIn *wire.TxIn, getPrevout func(wire.OutPoint) ([]byte, error)) (*btcec.PublicKey, error) {
	// P2SH
	if len(txIn.SignatureScript) > 0 && txscript.IsPayToWitnessPubKeyHash(txIn.SignatureScript[1:]) {
		if len(txIn.Witness) == 0 {
			return nil, ErrNonStandardScript
		}

		pubKeyBytes := txIn.Witness[len(txIn.Witness)-1]
		if len(pubKeyBytes) != 33 {
			return nil, ErrNonStandardScript
		}
		return btcec.ParsePubKey(pubKeyBytes)
	}

	// P2WPKH
	if len(txIn.Witness) == 2 {
		_, err := ecdsa.ParseSignature(txIn.Witness[1])
		if err == nil {
			if len(txIn.Witness[0]) != 33 {
				return nil, ErrNonStandardScript
			}

			return btcec.ParsePubKey(txIn.Witness[0])
		}
	}

	prevoutScript, err := getPrevout(txIn.PreviousOutPoint)
	if err != nil {
		return nil, err
	}
	// P2TR
	if txscript.IsPayToTaproot(prevoutScript) {
		witness := append(wire.TxWitness{}, txIn.Witness...)
		if len(witness) < 1 {
			return nil, ErrInvalidTaprootWitness
		}

		if len(witness) > 1 && len(witness[len(witness)-1]) > 0 && witness[len(witness)-1][0] == 0x50 {
			witness = witness[:len(witness)-1] // remove annex
		}

		if len(witness) > 1 {
			controlBlock := witness[len(witness)-1]
			internalKey := controlBlock[1:33]
			if bytes.Equal(internalKey, num_h) {
				return nil, ErrNonStandardScript
			}
		}

		taprootKey := prevoutScript[2:]
		return schnorr.ParsePubKey(taprootKey)
	}

	// P2PKH
	if txscript.IsPayToPubKeyHash(prevoutScript) {
		pubkeyHash := prevoutScript[3:23]
		for i := len(txIn.SignatureScript); i >= 0; i-- {
			if i-33 >= 0 {
				pubkey := txIn.SignatureScript[i-33 : i]

				if bytes.Equal(pubkeyHash, btcutil.Hash160(pubkey)) {
					return btcec.ParsePubKey(pubkey)
				}
			}
		}
	}

	return nil, ErrNonStandardScript
}
