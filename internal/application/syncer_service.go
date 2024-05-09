package application

import (
	"bytes"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/louisinger/silentiumd/internal/domain"
	"github.com/louisinger/silentiumd/internal/ports"
	"github.com/sirupsen/logrus"
)

var zeroHash, _ = chainhash.NewHashFromStr(
	"0000000000000000000000000000000000000000000000000000000000000000",
)

// SyncerService is running in background to sync the chain
// and compute silent payment scalars for each "send to taproot" transaction
// it is also responsible for storing the scalars in db and watching for spent taproot outputs
type SyncerService interface {
	Start() error
	Stop() error
}

type syncer struct {
	store       ports.ScalarRepository
	chainsource ports.ChainSource

	computeScalarsCh chan *btcutil.Block
	updateUnspentsCh chan *btcutil.Block

	stopUpdateUnspents      chan struct{}
	stopComputeBlockScalars chan struct{}
	stopSyncBlocks          chan struct{}
	stopBlockWatcher        chan struct{}
	startBlock              int32
}

func NewSyncerService(
	store ports.ScalarRepository,
	chainsrc ports.ChainSource,
	network chaincfg.Params,
	startBlock int32,
) (SyncerService, error) {
	start := startBlock

	latest, err := store.GetLatestBlockHeight()
	if err != nil {
		return nil, err
	}

	if latest > start {
		start = latest
	}

	// do not sync before taproot activation height
	if len(network.Deployments) > chaincfg.DeploymentTaproot {
		taprootHeight := int32(network.Deployments[chaincfg.DeploymentTaproot].MinActivationHeight)

		if taprootHeight > 0 && start < taprootHeight {
			start = taprootHeight
		}
	}

	logrus.Infof("start block: %d", start)

	return &syncer{store, chainsrc, nil, nil, nil, nil, nil, nil, int32(start)}, nil
}

func (s *syncer) Start() error {
	s.computeScalarsCh = make(chan *btcutil.Block)
	s.updateUnspentsCh = make(chan *btcutil.Block)

	s.stopBlockWatcher = make(chan struct{}, 1)
	s.stopSyncBlocks = make(chan struct{}, 1)
	s.stopUpdateUnspents = make(chan struct{}, 1)
	s.stopComputeBlockScalars = make(chan struct{}, 1)

	go func() {
		for {
			select {
			case block := <-s.computeScalarsCh:
				s.computeBlockScalars(block)
			case <-s.stopComputeBlockScalars:
				logrus.Info("stop compute block scalars")
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case block := <-s.updateUnspentsCh:
				s.updateUnspents(block)
			case <-s.stopUpdateUnspents:
				logrus.Info("stop update unspents")
				return
			}
		}
	}()

	go s.syncMissingBlocks()
	go s.blockWatcher()
	return nil
}

func (s *syncer) Stop() error {
	s.stopBlockWatcher <- struct{}{}
	s.stopSyncBlocks <- struct{}{}
	s.stopComputeBlockScalars <- struct{}{}
	s.stopUpdateUnspents <- struct{}{}

	close(s.computeScalarsCh)
	return nil
}

func (s *syncer) syncMissingBlocks() {
	tipHeight, err := s.chainsource.GetChainTipHeight()
	if err != nil {
		logrus.Error(err)
		return
	}

	latestHeight, err := s.store.GetLatestBlockHeight()
	if err != nil {
		logrus.Error(err)
		return
	}

	if latestHeight < s.startBlock {
		latestHeight = s.startBlock
	}

	if latestHeight < tipHeight {
		logrus.Infof("latest block height: %d, tip height: %d", latestHeight, tipHeight)
		logrus.Infof("syncing %d blocks", tipHeight-latestHeight)

		for i := int32(latestHeight + 1); i <= tipHeight; i++ {
			block, err := s.chainsource.GetBlockByHeight(i)
			if err != nil {
				logrus.Error(err)
				continue
			}

			select {
			case <-s.stopSyncBlocks:
				logrus.Info("stop sync blocks")
				return
			default:
				s.computeScalarsCh <- block
			}
		}
	}

}

func (s *syncer) blockWatcher() {
	blocksch, cancel, err := s.chainsource.SubscribeBlocks()
	if err != nil {
		return
	}

	for {
		select {
		case <-s.stopBlockWatcher:
			logrus.Info("stop block watcher")
			cancel()
			return
		case block := <-blocksch:
			logrus.Infof("new block %d", block.Height())
			s.computeScalarsCh <- block
		}
	}
}

func (s *syncer) computeBlockScalars(block *btcutil.Block) {
	t := time.Now()
	txs := block.Transactions()
	nbOfTxs := len(txs)

	scalars := make([]*domain.SilentScalar, 0)

	for i, tx := range txs {
		logrus.Debugf("tx %d/%d", i+1, nbOfTxs)
		if s.isSilentPaymentElligibleTx(tx) {
			scalar, err := domain.NewSilentScalar(tx)
			if err != nil {
				logrus.Error(err)
				return
			}

			if scalar == nil {
				return
			}

			if scalar.HasUnspentTaproot() {
				if err := scalar.ComputeScalar(s.chainsource.GetPrevoutScript); err != nil {
					logrus.Error(err)
					return
				}
				if scalar.Scalar != nil {
					scalars = append(scalars, scalar)
				}
			}
		}
	}

	if err := s.store.Write(scalars, block.Height()); err != nil {
		logrus.Error(err)
	}

	go func() {
		s.updateUnspentsCh <- block
	}()

	logrus.Infof("[%d] compute scalars done (%s)", block.Height(), time.Since(t))
}

func (s *syncer) updateUnspents(block *btcutil.Block) {
	var nbOfUpdates, nbOfDelete int

	for _, tx := range block.Transactions() {
		for _, input := range tx.MsgTx().TxIn {
			scalar, err := s.store.GetByTxHash(&input.PreviousOutPoint.Hash)
			if err != nil {
				if _, ok := err.(ports.ErrScalarNotFound); !ok {
					logrus.Error(err)
				}
				continue
			}

			scalar.MarkOutputSpent(input.PreviousOutPoint.Index)

			if !scalar.HasUnspentTaproot() {
				if err := s.store.Delete(&input.PreviousOutPoint.Hash); err != nil {
					logrus.Error(err)
				}
				nbOfDelete++
				continue
			}

			if err := s.store.Update(scalar); err != nil {
				logrus.Error(err)
			}
			nbOfUpdates++
		}
	}
	logrus.Infof("[%d] update done (%d updated, %d deleted)", block.Height(), nbOfUpdates, nbOfDelete)
}

// isSilentPaymentElligibleTx checks if a transaction is eligible for silent payments.
// it means that it must have at least 1 taproot output
func (s *syncer) isSilentPaymentElligibleTx(tx *btcutil.Tx) bool {
	for _, txIn := range tx.MsgTx().TxIn {
		// skip coinbase
		if txIn.PreviousOutPoint.Hash.IsEqual(zeroHash) {
			return false
		}

		if isInscription(txIn.Witness) {
			return false
		}
	}

	taprootOutputs := make(map[uint32][]byte)

	for i, txOut := range tx.MsgTx().TxOut {
		if txscript.IsPayToTaproot(txOut.PkScript) {
			taprootOutputs[uint32(i)] = txOut.PkScript
		}
	}

	return len(taprootOutputs) > 0
}

// admiting that the witness comes from taproot input,
// returns true if the tapscript is an inscription
// = OP_0 OP_IF .... OP_ENDIF
func isInscription(witness wire.TxWitness) bool {
	if len(witness) < 1 {
		return false
	}

	if len(witness) > 1 && len(witness[len(witness)-1]) > 0 && witness[len(witness)-1][0] == 0x50 {
		witness = witness[:len(witness)-1] // remove annex
	}

	if len(witness) < 2 {
		return false
	}

	tapscript := witness[len(witness)-2]

	ifIndex := bytes.IndexByte(tapscript, txscript.OP_IF)

	if ifIndex == -1 {
		return false
	}

	endifIndex := bytes.IndexByte(tapscript, txscript.OP_ENDIF)

	if endifIndex == -1 {
		return false
	}

	if ifIndex > endifIndex {
		return false
	}

	if ifIndex == 0 {
		return false
	}

	if tapscript[ifIndex-1] != txscript.OP_0 {
		return false
	}

	return true
}
