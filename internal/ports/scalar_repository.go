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
	GetByHeight(height int32) ([]*domain.SilentScalar, error)
	GetByTxHash(txHash *chainhash.Hash) (*domain.SilentScalar, error)
	Write(scalars []*domain.SilentScalar, blockHeight int32) error
	Delete(txhash *chainhash.Hash) error
	Update(scalar *domain.SilentScalar) error
}
