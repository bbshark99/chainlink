package vrf

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/smartcontractkit/chainlink/core/services/vrf/proof"

	"github.com/smartcontractkit/chainlink/core/gracefulpanic"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink/core/internal/gethwrappers/generated/vrf_coordinator_v2"
	"github.com/smartcontractkit/chainlink/core/logger"
	"github.com/smartcontractkit/chainlink/core/services/bulletprooftxmanager"
	"github.com/smartcontractkit/chainlink/core/services/eth"
	httypes "github.com/smartcontractkit/chainlink/core/services/headtracker/types"
	"github.com/smartcontractkit/chainlink/core/services/job"
	"github.com/smartcontractkit/chainlink/core/services/keystore"
	"github.com/smartcontractkit/chainlink/core/services/log"
	"github.com/smartcontractkit/chainlink/core/services/pipeline"
	"github.com/smartcontractkit/chainlink/core/services/postgres"
	"github.com/smartcontractkit/chainlink/core/store/models"
	"github.com/smartcontractkit/chainlink/core/utils"
	"go.uber.org/multierr"
	"gopkg.in/guregu/null.v4"
	"gorm.io/gorm"
)

const (
	// Gas to be used
	GasAfterPaymentCalculation = 5000 + // subID balance update
		2100 + // cold subscription balance read
		20000 + // first time oracle balance update, note first time will be 20k, but 5k subsequently
		2*2100 - // cold read oracle address and oracle balance
		4800 + // request delete refund, note pre-london fork was 15k
		21000 + // base cost of the transaction
		7748 // Static costs of argument encoding etc. note that it varies by +/- x*12 for every x bytes of non-zero data in the proof.
)

var (
	_ log.Listener = &listenerV2{}
	_ job.Service  = &listenerV2{}
)

type pendingRequest struct {
	confirmedAtBlock uint64
	req              *vrf_coordinator_v2.VRFCoordinatorV2RandomWordsRequested
	lb               log.Broadcast
}

type listenerV2 struct {
	utils.StartStopOnce
	cfg             Config
	l               logger.Logger
	abi             abi.ABI
	ethClient       eth.Client
	logBroadcaster  log.Broadcaster
	txm             bulletprooftxmanager.TxManager
	headBroadcaster httypes.HeadBroadcasterRegistry
	coordinator     *vrf_coordinator_v2.VRFCoordinatorV2
	pipelineRunner  pipeline.Runner
	pipelineORM     pipeline.ORM
	vorm            keystore.VRFORM
	job             job.Job
	db              *gorm.DB
	vrfks           *keystore.VRF
	gethks          *keystore.Eth
	mbLogs          *utils.Mailbox
	chStop          chan struct{}
	waitOnStop      chan struct{}
	latestHead      uint64
	// We can keep these pending logs in memory because we
	// only mark them confirmed once we send a corresponding fulfillment transaction.
	// So on node restart in the middle of processing, the lb will resend them.
	pendingLogs []pendingRequest
}

func (lsn *listenerV2) Start() error {
	return lsn.StartOnce("VRFListenerV2", func() error {
		// Take the larger of the global vs specific.
		// Note that the v2 vrf requests specify their own confirmation requirements.
		// We wait for max(minConfs, request required confs) to be safe.
		minConfs := lsn.cfg.MinIncomingConfirmations()
		if lsn.job.VRFSpec.Confirmations > lsn.cfg.MinIncomingConfirmations() {
			minConfs = lsn.job.VRFSpec.Confirmations
		}
		unsubscribeLogs := lsn.logBroadcaster.Register(lsn, log.ListenerOpts{
			Contract: lsn.coordinator.Address(),
			LogsWithTopics: map[common.Hash][][]log.Topic{
				vrf_coordinator_v2.VRFCoordinatorV2RandomWordsRequested{}.Topic(): {
					{
						log.Topic(lsn.job.VRFSpec.PublicKey.MustHash()),
					},
				},
			},
			// Do not specify min confirmations, as it varies from request to request.
		})

		// Subscribe to the head broadcaster for handling
		// per request conf requirements.
		_, unsubscribeHeadBroadcaster := lsn.headBroadcaster.Subscribe(lsn)

		go gracefulpanic.WrapRecover(func() {
			lsn.run([]func(){unsubscribeLogs, unsubscribeHeadBroadcaster}, minConfs)
		})
		return nil
	})
}

func (lsn *listenerV2) Connect(head *models.Head) error {
	lsn.latestHead = uint64(head.Number)
	return nil
}

func (lsn *listenerV2) OnNewLongestChain(ctx context.Context, head models.Head) {
	// Check if any v2 logs are ready for processing.
	lsn.latestHead = uint64(head.Number)
	var remainingLogs []pendingRequest
	for _, pl := range lsn.pendingLogs {
		if pl.confirmedAtBlock <= lsn.latestHead {
			// Note below makes API calls and opens a database transaction
			// TODO: Batch these requests in a follow up.
			lsn.ProcessV2VRFRequest(pl.req, pl.lb)
		} else {
			remainingLogs = append(remainingLogs, pl)
		}
	}
	lsn.pendingLogs = remainingLogs
}

func (lsn *listenerV2) run(unsubscribeLogs []func(), minConfs uint32) {
	lsn.l.Infow("VRFListenerV2: listening for run requests",
		"minConfs", minConfs)
	for {
		select {
		case <-lsn.chStop:
			for _, us := range unsubscribeLogs {
				us()
			}
			lsn.waitOnStop <- struct{}{}
			return
		case <-lsn.mbLogs.Notify():
			// Process all the logs in the queue if one is added
			for {
				i, exists := lsn.mbLogs.Retrieve()
				if !exists {
					break
				}
				lb, ok := i.(log.Broadcast)
				if !ok {
					panic(fmt.Sprintf("VRFListenerV2: invariant violated, expected log.Broadcast got %T", i))
				}
				alreadyConsumed, err := lsn.logBroadcaster.WasAlreadyConsumed(lsn.db, lb)
				if err != nil {
					lsn.l.Errorw("VRFListenerV2: could not determine if log was already consumed", "error", err, "txHash", lb.RawLog().TxHash)
					continue
				} else if alreadyConsumed {
					continue
				}
				req, err := lsn.coordinator.ParseRandomWordsRequested(lb.RawLog())
				if err != nil {
					lsn.l.Errorw("VRFListenerV2: failed to parse log", "err", err, "txHash", lb.RawLog().TxHash)
					lsn.markLogAsConsumed(lb)
					return
				}
				lsn.pendingLogs = append(lsn.pendingLogs, pendingRequest{
					confirmedAtBlock: req.Raw.BlockNumber + uint64(req.MinimumRequestConfirmations),
					req:              req,
					lb:               lb,
				})
			}
		}
	}
}

func (lsn *listenerV2) markLogAsConsumed(lb log.Broadcast) {
	ctx, cancel := postgres.DefaultQueryCtx()
	defer cancel()
	err := lsn.logBroadcaster.MarkConsumed(lsn.db.WithContext(ctx), lb)
	lsn.l.ErrorIf(errors.Wrapf(err, "VRFListenerV2: unable to mark log %v as consumed", lb.String()))
}

func (lsn *listenerV2) ProcessV2VRFRequest(req *vrf_coordinator_v2.VRFCoordinatorV2RandomWordsRequested, lb log.Broadcast) {
	// Check if the vrf req has already been fulfilled
	callback, err := lsn.coordinator.GetCommitment(nil, req.PreSeedAndRequestId)
	if err != nil {
		lsn.l.Errorw("VRFListenerV2: unable to check if already fulfilled, processing anyways", "err", err, "txHash", req.Raw.TxHash)
	} else if utils.IsEmpty(callback[:]) {
		// If seedAndBlockNumber is zero then the response has been fulfilled
		// and we should skip it
		lsn.l.Infow("VRFListenerV2: request already fulfilled", "txHash", req.Raw.TxHash, "subID", req.SubId, "callback", callback)
		lsn.markLogAsConsumed(lb)
		return
	}

	s := time.Now()
	proof, err1 := lsn.LogToProof(req, lb)
	vrfCoordinatorPayload, gasLimit, _, err2 := lsn.ProcessLogV2(proof)
	err = multierr.Combine(err1, err2)
	if err != nil {
		logger.Errorw("VRFListenerV2: error processing random request", "err", err, "txHash", req.Raw.TxHash)
	}
	f := time.Now()
	err = postgres.GormTransactionWithDefaultContext(lsn.db, func(tx *gorm.DB) error {
		if err == nil {
			// No errors processing the log, submit a transaction
			var etx bulletprooftxmanager.EthTx
			var from common.Address
			from, err = lsn.gethks.GetRoundRobinAddress()
			if err != nil {
				return err
			}
			etx, err = lsn.txm.CreateEthTransaction(tx,
				from,
				lsn.coordinator.Address(),
				vrfCoordinatorPayload,
				gasLimit,
				&models.EthTxMetaV2{
					JobID:         lsn.job.ID,
					RequestTxHash: lb.RawLog().TxHash,
				},
				bulletprooftxmanager.SendEveryStrategy{},
			)
			if err != nil {
				return err
			}
			// TODO: Once we have eth tasks supported, we can use the pipeline directly
			// and be able to save errored proof generations. Until then only save
			// successful runs and log errors.
			_, err = lsn.pipelineRunner.InsertFinishedRun(tx, pipeline.Run{
				State:          pipeline.RunStatusCompleted,
				PipelineSpecID: lsn.job.PipelineSpecID,
				Errors:         []null.String{{}},
				Outputs: pipeline.JSONSerializable{
					Val: []interface{}{fmt.Sprintf("queued tx from %v to %v txdata %v",
						etx.FromAddress,
						etx.ToAddress,
						hex.EncodeToString(etx.EncodedPayload))},
				},
				Meta: pipeline.JSONSerializable{
					Val: map[string]interface{}{"eth_tx_id": etx.ID},
				},
				CreatedAt:  s,
				FinishedAt: null.TimeFrom(f),
			}, nil, false)
			if err != nil {
				return errors.Wrap(err, "VRFListenerV2: failed to insert finished run")
			}
		}
		// Always mark consumed regardless of whether the proof failed or not.
		return lsn.logBroadcaster.MarkConsumed(tx, lb)
	})
	if err != nil {
		lsn.l.Errorw("VRFListenerV2 failed to save run", "err", err)
	}
}

func (lsn *listenerV2) LogToProof(req *vrf_coordinator_v2.VRFCoordinatorV2RandomWordsRequested, lb log.Broadcast) ([]byte, error) {
	lsn.l.Infow("VRFListenerV2: received log request",
		"log", lb.String(),
		"reqID", req.PreSeedAndRequestId.String(),
		"keyHash", hex.EncodeToString(req.KeyHash[:]),
		"txHash", req.Raw.TxHash,
		"blockNumber", req.Raw.BlockNumber,
		"seed", req.PreSeedAndRequestId.String())
	// Validate the key against the spec
	kh, err := lsn.job.VRFSpec.PublicKey.Hash()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(req.KeyHash[:], kh[:]) {
		return nil, fmt.Errorf("invalid key hash %v expected %v", hex.EncodeToString(req.KeyHash[:]), hex.EncodeToString(kh[:]))
	}

	// req.PreSeed is uint256(keccak256(abi.encode(keyHash, msg.sender, nonce)))
	preSeed, err := proof.BigToSeed(req.PreSeedAndRequestId)
	if err != nil {
		return nil, errors.New("unable to parse preseed")
	}
	seed := proof.PreSeedDataV2{
		PreSeed:          preSeed,
		BlockHash:        req.Raw.BlockHash,
		BlockNum:         req.Raw.BlockNumber,
		SubId:            req.SubId,
		CallbackGasLimit: req.CallbackGasLimit,
		NumWords:         req.NumWords,
		Sender:           req.Sender,
	}
	solidityProof, err := proof.GenerateProofResponseV2(lsn.vrfks, lsn.job.VRFSpec.PublicKey, seed)
	if err != nil {
		lsn.l.Errorw("VRFListenerV2: error generating proof", "err", err)
		return nil, err
	}
	return solidityProof[:], nil
}

func (lsn *listenerV2) ProcessLogV2(solidityProof []byte) ([]byte, uint64, *vrf_coordinator_v2.VRFCoordinatorV2RandomWordsRequested, error) {
	vrfCoordinatorArgs, err := lsn.abi.Methods["fulfillRandomWords"].Inputs.PackValues(
		[]interface{}{
			solidityProof[:], // geth expects slice, even if arg is constant-length
		})
	if err != nil {
		lsn.l.Errorw("VRFListenerV2: error building fulfill args", "err", err)
		return nil, 0, nil, err
	}
	to := lsn.coordinator.Address()
	gasLimit, err := lsn.ethClient.EstimateGas(context.Background(), ethereum.CallMsg{
		To:   &to,
		Data: append(lsn.abi.Methods["fulfillRandomWords"].ID, vrfCoordinatorArgs...),
	})
	if err != nil {
		lsn.l.Errorw("VRFListenerV2: error computing gas limit", "err", err)
		return nil, 0, nil, err
	}
	return append(lsn.abi.Methods["fulfillRandomWords"].ID, vrfCoordinatorArgs...), gasLimit, nil, nil
}

// Close complies with job.Service
func (lsn *listenerV2) Close() error {
	return lsn.StopOnce("VRFListenerV2", func() error {
		close(lsn.chStop)
		<-lsn.waitOnStop
		return nil
	})
}

func (lsn *listenerV2) HandleLog(lb log.Broadcast) {
	wasOverCapacity := lsn.mbLogs.Deliver(lb)
	if wasOverCapacity {
		logger.Error("VRFListenerV2: log mailbox is over capacity - dropped the oldest log")
	}
}

// Job complies with log.Listener
func (lsn *listenerV2) JobID() int32 {
	return lsn.job.ID
}