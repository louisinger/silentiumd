package domain_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/louisinger/echemythosd/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestNewSilentScalar(t *testing.T) {
	var testVectors []testVector
	openJSONFile(t, "test_data/test_vectors.json", &testVectors)

	for _, testVector := range testVectors {
		if testVector.Comment != "P2PKH and P2WPKH Uncompressed Keys are skipped" {
			continue
		}

		t.Run(testVector.Comment, func(t *testing.T) {
			for _, receiving := range testVector.Receiving {
				if len(receiving.Given.Labels) > 0 {
					continue
				}

				if len(receiving.Expected.Outputs) == 0 {
					continue
				}
				vin := make([]*wire.TxIn, 0, len(receiving.Given.Vin))

				for _, given := range receiving.Given.Vin {
					txIn := given.toWire()
					if txIn == nil {
						t.Fatalf("failed to create TxIn")
						return
					}

					vin = append(vin, txIn)
				}
				silentScalar := &domain.SilentScalar{
					TxIn: vin,
				}

				err := silentScalar.ComputeScalar(
					func(outpoint wire.OutPoint) ([]byte, error) {
						for _, given := range receiving.Given.Vin {
							if given.TxId == outpoint.Hash.String() && given.Vout == int(outpoint.Index) {
								return hex.DecodeString(given.Prevout.ScriptPubKey.Hex)
							}
						}

						return nil, errors.New("scriptPubKey not found")
					},
				)
				require.NoError(t, err)

				scalarPubKey, err := btcec.ParsePubKey(silentScalar.Scalar)
				require.NoError(t, err)

				scanPrvKeyBytes, err := hex.DecodeString(receiving.Given.KeyMaterial.ScanPrivKey)
				require.NoError(t, err)

				scanPrvKey, scanPublicKey := btcec.PrivKeyFromBytes(scanPrvKeyBytes)
				require.NotNil(t, scanPrvKey)
				require.NotNil(t, scanPublicKey)

				secret := ecdhSharedSecret(scanPrvKey, scalarPubKey)
				require.NotNil(t, secret)

				secretBytes := secret.SerializeCompressed()

				expectedPubKeys := make([]string, 0, len(receiving.Expected.Outputs))
				for _, output := range receiving.Expected.Outputs {
					expectedPubKeys = append(expectedPubKeys, output.PubKey)
				}

				k := uint32(0)
				for len(expectedPubKeys) > 0 {
					hash := chainhash.TaggedHash(
						[]byte("BIP0352/SharedSecret"),
						append(secretBytes, serUint32(k)...),
					)

					spendPrvKeyByes, err := hex.DecodeString(receiving.Given.KeyMaterial.SpendPrivKey)
					require.NoError(t, err)
					_, spendPublicKey := btcec.PrivKeyFromBytes(spendPrvKeyByes)
					require.NotNil(t, spendPublicKey)

					xScalar, yScalar := btcec.S256().ScalarBaseMult(hash[:])
					x, y := btcec.S256().Add(spendPublicKey.X(), spendPublicKey.Y(), xScalar, yScalar)
					var xFieldVal, yFieldVal btcec.FieldVal
					xFieldVal.SetByteSlice(x.Bytes())
					yFieldVal.SetByteSlice(y.Bytes())

					resultAsKey := btcec.NewPublicKey(&xFieldVal, &yFieldVal)
					tapKey := schnorr.SerializePubKey(resultAsKey)

					require.Contains(
						t,
						expectedPubKeys,
						hex.EncodeToString(tapKey),
						"k ="+fmt.Sprint(k),
					)

					for i, expectedPubKey := range expectedPubKeys {
						if expectedPubKey == hex.EncodeToString(tapKey) {
							expectedPubKeys = append(expectedPubKeys[:i], expectedPubKeys[i+1:]...)
							break
						}
					}
					k += 1
				}
			}
		})
	}
}

func serUint32(i uint32) []byte {
	var index [4]byte
	binary.BigEndian.PutUint32(index[:], i)

	return index[:]
}

func decodeWitness(witness []byte) wire.TxWitness {
	reader := bytes.NewReader(witness)

	len, err := wire.ReadVarInt(reader, 0)
	if err != nil {
		return nil
	}

	wit := make(wire.TxWitness, len)
	for i := range wit {
		wit[i], err = readScript(reader, 0)
		if err != nil {
			panic(err)
		}
	}

	return wit
}

func readScript(r io.Reader, pver uint32) ([]byte, error) {
	count, err := wire.ReadVarInt(r, pver)
	if err != nil {
		return nil, err
	}

	b := make([]byte, count)
	_, err = io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func openJSONFile(t *testing.T, path string, result interface{}) {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(result); err != nil {
		t.Fatalf("failed to decode file: %v", err)
	}
}

type inputVector struct {
	TxId        string `json:"txid"`
	Vout        int    `json:"vout"`
	ScriptSig   string `json:"scriptSig"`
	TxInWitness string `json:"txinwitness"`
	Prevout     struct {
		ScriptPubKey struct {
			Hex string `json:"hex"`
		} `json:"scriptPubKey"`
	} `json:"prevout"`
	PrivateKey string `json:"privateKey"`
}

func (iv *inputVector) toWire() *wire.TxIn {
	txid, _ := chainhash.NewHashFromStr(iv.TxId)
	signatureScript, _ := hex.DecodeString(iv.ScriptSig)
	witnessEncoded, _ := hex.DecodeString(iv.TxInWitness)

	return &wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  *txid,
			Index: uint32(iv.Vout),
		},
		SignatureScript: signatureScript,
		Witness:         decodeWitness(witnessEncoded),
		Sequence:        wire.MaxTxInSequenceNum,
	}
}

type testVector struct {
	Comment string `json:"comment"`
	Sending []struct {
		Given struct {
			Vin        []inputVector `json:"vin"`
			Recipients []string      `json:"recipients"`
		} `json:"given"`
		Expected struct {
			Outputs []string `json:"outputs"`
		} `json:"expected"`
	} `json:"sending"`
	Receiving []struct {
		Given struct {
			Vin         []inputVector `json:"vin"`
			Outputs     []string      `json:"outputs"`
			KeyMaterial struct {
				SpendPrivKey string `json:"spend_priv_key"`
				ScanPrivKey  string `json:"scan_priv_key"`
			} `json:"key_material"`
			Labels []int `json:"labels"`
		} `json:"given"`
		Expected struct {
			Outputs []struct {
				PubKey       string `json:"pub_key"`
				PrivKeyTweak string `json:"priv_key_tweak"`
				Signature    string `json:"signature"`
			} `json:"outputs"`
		} `json:"expected"`
	} `json:"receiving"`
}

func ecdhSharedSecret(privkey *btcec.PrivateKey, pubkey *btcec.PublicKey) *btcec.PublicKey {
	var point, result btcec.JacobianPoint
	pubkey.AsJacobian(&point)
	btcec.ScalarMultNonConst(&privkey.Key, &point, &result)
	result.ToAffine()
	return btcec.NewPublicKey(&result.X, &result.Y)
}
