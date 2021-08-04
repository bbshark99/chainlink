package offchainreporting2

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink/core/services/log"
	"github.com/smartcontractkit/libocr/gethwrappers2/offchainaggregator"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2/types"
)

var (
	_ ocrtypes.ContractTransmitter = &OCRContractTransmitter{}
)

type (
	OCRContractTransmitter struct {
		contractAddress gethCommon.Address
		contractABI     abi.ABI
		transmitter     Transmitter
		contractCaller  *offchainaggregator.OffchainAggregatorCaller
		tracker         *OCRContractTracker
		chainID         *big.Int
	}

	Transmitter interface {
		CreateEthTransaction(ctx context.Context, toAddress gethCommon.Address, payload []byte) error
		FromAddress() gethCommon.Address
	}
)

func NewOCRContractTransmitter(
	address gethCommon.Address,
	contractCaller *offchainaggregator.OffchainAggregatorCaller,
	contractABI abi.ABI,
	transmitter Transmitter,
	logBroadcaster log.Broadcaster,
	tracker *OCRContractTracker,
	chainID *big.Int,
) *OCRContractTransmitter {
	return &OCRContractTransmitter{
		contractAddress: address,
		contractABI:     contractABI,
		transmitter:     transmitter,
		contractCaller:  contractCaller,
		tracker:         tracker,
		chainID:         chainID,
	}
}

func (oc *OCRContractTransmitter) Transmit(ctx context.Context, reportCtx ocrtypes.ReportContext, report ocrtypes.Report, signatures []ocrtypes.AttributedOnChainSignature) error {
	payload, err := oc.contractABI.Pack("transmit", reportCtx, report, signatures)
	if err != nil {
		return errors.Wrap(err, "abi.Pack failed")
	}

	return errors.Wrap(oc.transmitter.CreateEthTransaction(ctx, oc.contractAddress, payload), "failed to send Eth transaction")
}

func (oc *OCRContractTransmitter) FromAddress() gethCommon.Address {
	return oc.transmitter.FromAddress()
}

func (oc *OCRContractTransmitter) ChainID() *big.Int {
	return oc.chainID
}

func (oc *OCRContractTransmitter) LatestConfigDigestAndEpoch(ctx context.Context) (ocrtypes.ConfigDigest, uint32, error) {
	opts := bind.CallOpts{Context: ctx, Pending: false}
	result, err := oc.contractCaller.LatestTransmissionDetails(&opts)
	if err != nil {
		return ocrtypes.ConfigDigest{}, 0, errors.Wrap(err, "error getting LatestTransmissionDetails")
	}
	return result.ConfigDigest, result.Epoch, nil
}

func (oc *OCRContractTransmitter) FromAccount() ocrtypes.Account {
	return ocrtypes.Account("")
}
