package dbtest

import (
	"crypto/rand"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/louisinger/echemythosd/internal/domain"
	badgerdb "github.com/louisinger/echemythosd/internal/infrastructure/db/badger"
	"github.com/louisinger/echemythosd/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestGetLatestBlockHeight(t *testing.T) {
	repositories := getRepositories(t)
	for name, repo := range repositories {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, repo.Write([]*domain.SilentScalar{}, 10000))

			latest, err := repo.GetLatestBlockHeight()
			require.NoError(t, err)
			require.Equal(t, int32(10000), latest)

			require.NoError(t, repo.Write([]*domain.SilentScalar{}, 10001))

			latest, err = repo.GetLatestBlockHeight()
			require.NoError(t, err)
			require.Equal(t, int32(10001), latest)
		})
	}
}

func TestGetByHeight(t *testing.T) {
	repositories := getRepositories(t)
	for name, repo := range repositories {
		t.Run(name, func(t *testing.T) {
			txhash := generateRandomTxHash(t)
			require.NoError(t, repo.Write([]*domain.SilentScalar{
				{
					TaprootOutputs: []domain.TaprootOutput{
						{
							Index: 0,
							Spent: false,
						},
					},
					Scalar: []byte{0x01},
					TxHash: txhash,
				},
			}, 10000))

			scalars, err := repo.GetByHeight(10000)
			require.NoError(t, err)
			require.Len(t, scalars, 1)
			require.Equal(t, []domain.TaprootOutput{
				{
					Index: 0,
					Spent: false,
				},
			}, scalars[0].TaprootOutputs)
			require.Equal(t, []byte{0x01}, scalars[0].Scalar)
			require.Equal(t, txhash, scalars[0].TxHash)
		})
	}
}

func TestGetByTxHash(t *testing.T) {
	repositories := getRepositories(t)
	for name, repo := range repositories {
		t.Run(name, func(t *testing.T) {
			txHash := generateRandomTxHash(t)

			require.NoError(t, repo.Write([]*domain.SilentScalar{
				{
					TaprootOutputs: []domain.TaprootOutput{
						{
							Index: 0,
							Spent: false,
						},
					},
					Scalar: []byte{0x01},
					TxHash: txHash,
				},
			}, 10000))

			scalar, err := repo.GetByTxHash(txHash)
			require.NoError(t, err)
			require.Equal(t, []domain.TaprootOutput{
				{
					Index: 0,
					Spent: false,
				},
			}, scalar.TaprootOutputs)
			require.Equal(t, []byte{0x01}, scalar.Scalar)
			require.Equal(t, txHash, scalar.TxHash)
		})
	}
}

func TestDelete(t *testing.T) {
	repositories := getRepositories(t)
	for name, repo := range repositories {
		t.Run(name, func(t *testing.T) {
			txHash := generateRandomTxHash(t)

			require.NoError(t, repo.Write([]*domain.SilentScalar{
				{
					TaprootOutputs: []domain.TaprootOutput{
						{
							Index: 0,
							Spent: false,
						},
					},
					Scalar: []byte{0x01},
					TxHash: txHash,
				},
			}, 10000))

			err := repo.Delete(txHash)
			require.NoError(t, err)

			_, err = repo.GetByTxHash(txHash)
			require.Error(t, err)
			require.Equal(t, ports.ErrScalarNotFound{MethodName: "GetByTxHash"}, err)
		})
	}
}

func TestUpdate(t *testing.T) {
	repositories := getRepositories(t)
	for name, repo := range repositories {
		t.Run(name, func(t *testing.T) {
			txHash := generateRandomTxHash(t)

			require.NoError(t, repo.Write([]*domain.SilentScalar{
				{
					TaprootOutputs: []domain.TaprootOutput{
						{
							Index: 0,
							Spent: false,
						},
					},
					Scalar: []byte{0x01},
					TxHash: txHash,
				},
			}, 10000))

			scalar, err := repo.GetByTxHash(txHash)
			require.NoError(t, err)
			require.Equal(t, []domain.TaprootOutput{
				{
					Index: 0,
					Spent: false,
				},
			}, scalar.TaprootOutputs)
			require.Equal(t, []byte{0x01}, scalar.Scalar)
			require.Equal(t, txHash, scalar.TxHash)

			scalar.Scalar = []byte{0x02}
			scalar.TaprootOutputs[0].Spent = true

			err = repo.Update(scalar)
			require.NoError(t, err)

			scalar, err = repo.GetByTxHash(txHash)
			require.NoError(t, err)
			require.Equal(t, []domain.TaprootOutput{
				{
					Index: 0,
					Spent: true,
				},
			}, scalar.TaprootOutputs)
			require.Equal(t, []byte{0x02}, scalar.Scalar)
			require.Equal(t, txHash, scalar.TxHash)
		})
	}

}

func getRepositories(t *testing.T) map[string]ports.ScalarRepository {
	badgerrepo, err := badgerdb.NewScalarRepository("", nil)
	require.NoError(t, err)

	return map[string]ports.ScalarRepository{
		"badger": badgerrepo,
	}
}

func generateRandomTxHash(t *testing.T) *chainhash.Hash {
	random32bytes := make([]byte, 32)
	_, err := rand.Read(random32bytes)
	require.NoError(t, err)

	hash, err := chainhash.NewHash(random32bytes)
	require.NoError(t, err)
	return hash
}
