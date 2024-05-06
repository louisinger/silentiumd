package ports

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
)

type ChainSource interface {
	GetPrevoutScript(wire.OutPoint) ([]byte, error)
	SubscribeBlocks() (<-chan *btcutil.Block, func(), error)
	GetChainTipHeight() (int32, error)
	GetBlockByHeight(int32) (*btcutil.Block, error)
	GetBlockFilterByHeight(int32) (string, string, error)
	IsUtxo(outpoint wire.OutPoint) (bool, error)
}
