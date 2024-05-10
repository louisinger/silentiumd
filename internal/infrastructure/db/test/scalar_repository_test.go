package dbtest

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/louisinger/silentiumd/internal/domain"
	badgerdb "github.com/louisinger/silentiumd/internal/infrastructure/db/badger"
	"github.com/louisinger/silentiumd/internal/infrastructure/db/postgres"
	"github.com/louisinger/silentiumd/internal/ports"
	"github.com/stretchr/testify/require"
)

const (
	testDSN = "postgres://postgres:admin@localhost:5432?sslmode=disable"
)

func TestGetLatestBlockHeight(t *testing.T) {
	repositories := getRepositories(t)
	for name, repo := range repositories {
		t.Run(name, func(t *testing.T) {
			initialTip, err := repo.GetLatestBlockHeight()
			require.NoError(t, err)

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
			}, initialTip+1))

			latest, err := repo.GetLatestBlockHeight()
			require.NoError(t, err)
			require.Equal(t, initialTip+1, latest)

			txhash2 := generateRandomTxHash(t)
			require.NoError(t, repo.Write([]*domain.SilentScalar{
				{
					TaprootOutputs: []domain.TaprootOutput{
						{
							Index: 0,
							Spent: false,
						},
					},
					Scalar: []byte{0x02},
					TxHash: txhash2,
				},
			}, initialTip+2))

			latest, err = repo.GetLatestBlockHeight()
			require.NoError(t, err)
			require.Equal(t, initialTip+2, latest)
		})
	}
}

func TestGetScalars(t *testing.T) {
	repositories := getRepositories(t)
	for name, repo := range repositories {
		t.Run(name, func(t *testing.T) {
			txhash := generateRandomTxHash(t)
			blockHeight := randomBlockHeight(t)
			require.NoError(t, repo.Write([]*domain.SilentScalar{
				{
					TaprootOutputs: []domain.TaprootOutput{
						{
							Index: 0,
							Spent: false,
						},
						{
							Index: 1,
							Spent: false,
						},
					},
					Scalar: []byte{0x03},
					TxHash: txhash,
				},
			}, blockHeight))

			scalars, err := repo.GetScalars(blockHeight)
			require.NoError(t, err)
			require.Len(t, scalars, 1)
			require.Equal(t, hex.EncodeToString([]byte{0x03}), scalars[0])

			err = repo.MarkOutpointSpent(txhash, 0)
			require.NoError(t, err)

			scalars, err = repo.GetScalars(blockHeight)
			require.NoError(t, err)
			require.Len(t, scalars, 1)

			err = repo.MarkOutpointSpent(txhash, 1)
			require.NoError(t, err)

			scalars, err = repo.GetScalars(blockHeight)
			require.NoError(t, err)
			require.Len(t, scalars, 0)
		})
	}
}

func getRepositories(t *testing.T) map[string]ports.ScalarRepository {
	badgerrepo, err := badgerdb.New("", nil)
	require.NoError(t, err)

	postresrepo, err := postgres.New(postgres.PostreSQLConfig{Dsn: testDSN})
	require.NoError(t, err)

	return map[string]ports.ScalarRepository{
		"badger":   badgerrepo,
		"postgres": postresrepo,
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

func randomBlockHeight(t *testing.T) int32 {
	random32bytes := make([]byte, 4)
	_, err := rand.Read(random32bytes)
	require.NoError(t, err)

	return int32(random32bytes[0])
}
