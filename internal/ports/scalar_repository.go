package ports

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/louisinger/silentiumd/internal/domain"
)

type ErrScalarNotFound struct {
	MethodName string
}

func (e ErrScalarNotFound) Error() string {
	return fmt.Sprintf("scalar not found (%s)", e.MethodName)
}

type ScalarRepository interface {
	GetLatestBlockHeight() (int32, error)
	GetScalars(height int32) ([]string, error)
	MarkOutpointSpent(txHash *chainhash.Hash, index uint32) error
	Write(scalars []*domain.SilentScalar, blockHeight int32) error
}
