package application

import (
	"github.com/louisinger/silentiumd/internal/domain"
	"github.com/louisinger/silentiumd/internal/ports"
)

type SilentiumService interface {
	GetScalarsByHeight(height uint32) ([]*domain.SilentScalar, error)
	GetBlockFilter(height uint32) (filter string, header string, err error)
	GetChainTip() (uint32, error)
}

type silentium struct {
	repo        ports.ScalarRepository
	chainsource ports.ChainSource
}

func NewSilentiumService(repo ports.ScalarRepository, chainsource ports.ChainSource) SilentiumService {
	return &silentium{repo, chainsource}
}

func (e *silentium) GetChainTip() (uint32, error) {
	last, err := e.repo.GetLatestBlockHeight()
	if err != nil {
		return 0, err
	}

	return uint32(last), nil
}

func (e *silentium) GetScalarsByHeight(height uint32) ([]*domain.SilentScalar, error) {
	return e.repo.GetByHeight(int32(height))
}

func (e *silentium) GetBlockFilter(height uint32) (filter string, blockhash string, err error) {
	return e.chainsource.GetBlockFilterByHeight(int32(height))
}
