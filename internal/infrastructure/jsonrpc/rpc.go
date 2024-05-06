package jsonrpc

import (
	"bytes"
	"errors"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/louisinger/echemythosd/internal/ports"
	"github.com/sirupsen/logrus"
)

var blockFilterType = btcjson.FilterTypeBasic

type clientRPC struct {
	rpc *rpcclient.Client
}

func New(host, cookiePath string) (*clientRPC, error) {
	rpc, err := rpcclient.New(&rpcclient.ConnConfig{
		Host:         host,
		CookiePath:   cookiePath,
		DisableTLS:   true,
		HTTPPostMode: true,
	}, nil)
	if err != nil {
		return nil, err
	}

	return &clientRPC{rpc}, nil
}

func NewUnsafe(host, user, pass string) (*clientRPC, error) {
	rpc, err := rpcclient.New(&rpcclient.ConnConfig{
		Host:         host,
		User:         user,
		Pass:         pass,
		DisableTLS:   true,
		HTTPPostMode: true,
	}, nil)
	if err != nil {
		return nil, err
	}

	return &clientRPC{rpc}, nil
}

var _ ports.ChainSource = &clientRPC{}

func (c *clientRPC) GetBlockByHeight(h int32) (*btcutil.Block, error) {
	hash, err := c.rpc.GetBlockHash(int64(h))
	if err != nil {
		return nil, err
	}

	block, err := c.rpc.GetBlock(hash)
	if err != nil {
		return nil, err
	}

	var serializedBlock bytes.Buffer
	if err := block.Serialize(&serializedBlock); err != nil {
		return nil, err
	}

	b := btcutil.NewBlockFromBlockAndBytes(block, serializedBlock.Bytes())
	b.SetHeight(h)
	return b, nil
}

func (c *clientRPC) GetBlockFilterByHeight(h int32) (string, string, error) {
	hash, err := c.rpc.GetBlockHash(int64(h))
	if err != nil {
		logrus.Error("GetBlockHash failed: ", err)
		return "", "", err
	}

	filter, err := c.rpc.GetBlockFilter(*hash, &blockFilterType)
	if err != nil {
		logrus.Error("GetCFilter failed: ", err)
		return "", "", err
	}

	return filter.Filter, hash.String(), nil
}

func (c *clientRPC) GetChainTipHeight() (int32, error) {
	info, err := c.rpc.GetBlockChainInfo()
	if err != nil {
		return 0, err
	}

	return info.Blocks, nil
}

func (c *clientRPC) GetPrevoutScript(outpoint wire.OutPoint) ([]byte, error) {
	tx, err := c.rpc.GetRawTransaction(&outpoint.Hash)
	if err != nil {
		return nil, err
	}

	if int(outpoint.Index) >= len(tx.MsgTx().TxOut) {
		return nil, errors.New("index out of range")
	}

	return tx.MsgTx().TxOut[outpoint.Index].PkScript, nil
}

func (c *clientRPC) HasOneUnspent(txhash chainhash.Hash, outputsPkScript map[uint32][]byte, startBlock int32) (bool, error) {
	panic("unimplemented")
}

func (c *clientRPC) SubscribeBlocks() (<-chan *btcutil.Block, func(), error) {
	currentHeight, err := c.GetChainTipHeight()
	if err != nil {
		return nil, nil, err
	}

	ticker := time.NewTicker(1 * time.Minute)
	quit := make(chan struct{})

	blockChan := make(chan *btcutil.Block)

	go func() {
		defer close(blockChan)
		defer ticker.Stop()

		for {
			select {
			case <-quit:
				return
			case <-ticker.C:
				newHeight, err := c.GetChainTipHeight()
				if err != nil {
					logrus.Error(err)
					continue
				}

				if newHeight > currentHeight {
					for h := currentHeight + 1; h <= newHeight; h++ {
						block, err := c.GetBlockByHeight(h)
						if err != nil {
							logrus.Error(err)
							continue
						}

						blockChan <- block
					}
					currentHeight = newHeight
				}
			}
		}
	}()

	return blockChan, func() {
		quit <- struct{}{}
		close(quit)
	}, nil
}

func (c *clientRPC) IsUtxo(outpoint wire.OutPoint) (bool, error) {
	res, err := c.rpc.GetTxOut(&outpoint.Hash, outpoint.Index, false)
	if err != nil {
		return false, err
	}

	return res != nil, nil
}
