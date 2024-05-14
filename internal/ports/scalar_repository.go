package ports

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/louisinger/silentiumd/internal/domain"
)

type ErrScalarNotFound struct {
	MethodName string
}

type Outpoint struct {
	TxHash *chainhash.Hash
	Index  uint32
}

func (e ErrScalarNotFound) Error() string {
	return fmt.Sprintf("scalar not found (%s)", e.MethodName)
}

type ScalarRepository interface {
	GetLatestBlockHeight() (int32, error)
	GetScalars(height int32) ([]string, error)
	MarkSpent(outpoints []wire.OutPoint) error
	Write(scalars []*domain.SilentScalar, blockHeight int32) error
}
