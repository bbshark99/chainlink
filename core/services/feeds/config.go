package feeds

import (
	"math/big"
	"time"

	"github.com/smartcontractkit/chainlink/core/store/models"
)

//go:generate mockery --name Config --output ./mocks/ --case=underscore

type Config interface {
	ChainID() *big.Int
	Dev() bool
	DefaultHTTPTimeout() models.Duration

	FeatureOffchainReporting() bool
	OCRBlockchainTimeout(override time.Duration) time.Duration
	OCRContractConfirmations(override uint16) uint16
	OCRContractPollInterval(override time.Duration) time.Duration
	OCRContractSubscribeInterval(override time.Duration) time.Duration
	OCRContractTransmitterTransmitTimeout() time.Duration
	OCRDatabaseTimeout() time.Duration
	OCRObservationTimeout(override time.Duration) time.Duration
	OCRObservationGracePeriod() time.Duration

	FeatureOffchainReporting2() bool
	OCR2BlockchainTimeout(override time.Duration) time.Duration
	OCR2ContractConfirmations(override uint16) uint16
	OCR2ContractPollInterval(override time.Duration) time.Duration
	OCR2ContractSubscribeInterval(override time.Duration) time.Duration
	OCR2ContractTransmitterTransmitTimeout() time.Duration
	OCR2DatabaseTimeout() time.Duration
	OCR2ObservationTimeout(override time.Duration) time.Duration
	OCR2ObservationGracePeriod() time.Duration
}
