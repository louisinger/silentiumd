package badgerdb

import (
	"encoding/hex"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/louisinger/silentiumd/internal/domain"
	"github.com/louisinger/silentiumd/internal/ports"
	"github.com/sirupsen/logrus"
	"github.com/timshannon/badgerhold/v4"
)

const (
	globalKey = "global"
)

type scalarRepository struct {
	store *badgerhold.Store
}

func NewScalarRepository(
	baseDir string,
	logger badger.Logger,
) (ports.ScalarRepository, error) {
	logrus.Warn("using badgerdb, consider using postgresql for production")

	db, err := createDb(baseDir, logger)
	if err != nil {
		return nil, err
	}

	return &scalarRepository{db}, nil
}

func (s *scalarRepository) GetScalars(height int32) ([]string, error) {
	var result blockScalarsDTO
	if err := s.store.Get(height, &result); err != nil {
		return nil, err
	}

	scalars := make([]string, 0, len(result.ScalarsData))
	for _, scalar := range result.ScalarsData {
		scalars = append(scalars, hex.EncodeToString(scalar.Scalar))
	}

	return scalars, nil
}

func (s *scalarRepository) MarkOutpointSpent(txHash *chainhash.Hash, index uint32) error {
	silentScalar, err := s.getByTxHash(txHash)
	if err != nil {
		return err
	}

	atLeastOneUnspent := false

	for i, out := range silentScalar.TaprootOutputs {
		if out.Index == index {
			silentScalar.TaprootOutputs[i].Spent = true
			continue
		}

		if !out.Spent {
			atLeastOneUnspent = true
		}
	}

	if atLeastOneUnspent {
		return s.update(silentScalar)
	}

	return s.delete(txHash)
}

func (s *scalarRepository) delete(txHash *chainhash.Hash) error {
	var result blockScalarsDTO

	if err := s.store.FindOne(&result, badgerhold.Where("ScalarsData").HasKey(*txHash)); err != nil {
		if err == badgerhold.ErrNotFound {
			return ports.ErrScalarNotFound{MethodName: "Delete"}
		}

		return err
	}

	delete(result.ScalarsData, *txHash)

	return s.store.Update(result.Height, &result)
}

func (s *scalarRepository) getByTxHash(txHash *chainhash.Hash) (*domain.SilentScalar, error) {
	var result blockScalarsDTO

	if err := s.store.FindOne(&result, badgerhold.Where("ScalarsData").HasKey(*txHash)); err != nil {
		if err == badgerhold.ErrNotFound {
			return nil, ports.ErrScalarNotFound{MethodName: "GetByTxHash"}
		}

		return nil, err
	}

	scalarData, ok := result.ScalarsData[*txHash]
	if !ok {
		return nil, ports.ErrScalarNotFound{MethodName: "GetByTxHash"}
	}

	return &domain.SilentScalar{
		TxHash:         txHash,
		TaprootOutputs: scalarData.TaprootOutputs,
		Scalar:         scalarData.Scalar,
	}, nil
}

func (s *scalarRepository) GetLatestBlockHeight() (int32, error) {
	var result global

	if err := s.store.Get(globalKey, &result); err != nil {
		if err == badgerhold.ErrNotFound {
			return 0, nil
		}

		return 0, err
	}

	return result.MaxHeight, nil
}

func (s *scalarRepository) Write(scalars []*domain.SilentScalar, blockHeight int32) error {
	if err := s.store.Upsert(blockHeight, newDTO(blockHeight, scalars)); err != nil {
		return err
	}

	maxHeight, err := s.GetLatestBlockHeight()
	if err != nil {
		return err
	}

	if blockHeight > maxHeight {
		if err := s.store.Upsert(globalKey, global{blockHeight}); err != nil {
			return err
		}
	}

	return nil
}

func (s *scalarRepository) update(updated *domain.SilentScalar) error {
	var result blockScalarsDTO

	if err := s.store.FindOne(&result, badgerhold.Where("ScalarsData").HasKey(*updated.TxHash)); err != nil {
		if err == badgerhold.ErrNotFound {
			return ports.ErrScalarNotFound{MethodName: "Update"}
		}

		return err
	}

	result.ScalarsData[*updated.TxHash] = scalar{
		TaprootOutputs: updated.TaprootOutputs,
		Scalar:         updated.Scalar,
	}

	return s.store.Update(result.Height, &result)
}

func createDb(dbDir string, logger badger.Logger) (*badgerhold.Store, error) {
	isInMemory := len(dbDir) <= 0

	opts := badger.DefaultOptions(dbDir)
	opts.Logger = logger

	if isInMemory {
		opts.InMemory = true
	} else {
		opts.Compression = options.ZSTD
	}

	db, err := badgerhold.Open(badgerhold.Options{
		Encoder:          badgerhold.DefaultEncode,
		Decoder:          badgerhold.DefaultDecode,
		SequenceBandwith: 100,
		Options:          opts,
	})
	if err != nil {
		return nil, err
	}

	if !isInMemory {
		ticker := time.NewTicker(30 * time.Minute)

		go func() {
			for {
				<-ticker.C
				if err := db.Badger().RunValueLogGC(0.5); err != nil && err != badger.ErrNoRewrite {
					logrus.Error(err)
				}
			}
		}()
	}

	return db, nil
}
