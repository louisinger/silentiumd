package badgerdb

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/louisinger/silentiumd/internal/domain"
)

type global struct {
	MaxHeight int32
}

type scalar struct {
	Scalar         []byte
	TaprootOutputs []domain.TaprootOutput
}

type blockScalarsDTO struct {
	Height      int32 `badgerhold:"key"`
	ScalarsData map[chainhash.Hash]scalar
}

func newDTO(height int32, scalars []*domain.SilentScalar) *blockScalarsDTO {
	scalarsData := make(map[chainhash.Hash]scalar, len(scalars))
	for _, s := range scalars {
		scalarsData[*s.TxHash] = scalar{
			Scalar:         s.Scalar,
			TaprootOutputs: s.TaprootOutputs,
		}
	}
	return &blockScalarsDTO{
		Height:      height,
		ScalarsData: scalarsData,
	}

}

func (b *blockScalarsDTO) Scalars() []*domain.SilentScalar {
	scalars := make([]*domain.SilentScalar, 0, len(b.ScalarsData))
	for txHash, scalar := range b.ScalarsData {
		scalars = append(scalars, &domain.SilentScalar{
			TxHash:         &txHash,
			TaprootOutputs: scalar.TaprootOutputs,
			Scalar:         scalar.Scalar,
		})
	}
	return scalars
}
