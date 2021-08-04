package config

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink/core/logger"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/ethkey"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/p2pkey"
	"github.com/smartcontractkit/chainlink/core/store/models"
	ocrcommontypes "github.com/smartcontractkit/libocr/commontypes"
	ocrnetworking "github.com/smartcontractkit/libocr/networking"
)

// OCR2P2PListenIP is the ip that libp2p willl bind to and listen on
func (c Config) OCR2P2PListenIP() net.IP {
	return c.getWithFallback("OCR2P2PListenIP", parseIP).(net.IP)
}

// OCR2P2PListenPort is the port that libp2p will bind to and listen on
func (c *Config) OCR2P2PListenPort() uint16 {
	if c.viper.IsSet(EnvVarName("OCR2P2PListenPort")) {
		return uint16(c.viper.GetUint32(EnvVarName("OCR2P2PListenPort")))
	}
	// Fast path in case it was already set
	c.randomOCR2P2PPortMtx.RLock()
	if c.randomOCR2P2PPort > 0 {
		c.randomOCR2P2PPortMtx.RUnlock()
		return c.randomOCR2P2PPort
	}
	c.randomOCR2P2PPortMtx.RUnlock()
	// Path for initial set
	c.randomOCR2P2PPortMtx.Lock()
	defer c.randomOCR2P2PPortMtx.Unlock()
	if c.randomOCR2P2PPort > 0 {
		return c.randomOCR2P2PPort
	}
	r, err := rand.Int(rand.Reader, big.NewInt(65535-1023))
	if err != nil {
		logger.Fatalw("unexpected error generating random port", "err", err)
	}
	randPort := uint16(r.Int64() + 1024)
	logger.Warnw(fmt.Sprintf("OCR2_P2P_LISTEN_PORT was not set, listening on random port %d. A new random port will be generated on every boot, for stability it is recommended to set OCR2_P2P_LISTEN_PORT to a fixed value in your environment", randPort), "p2pPort", randPort)
	c.randomOCR2P2PPort = randPort
	return c.randomOCR2P2PPort
}

// OCR2P2PListenPortRaw returns the raw string value of OCR2_P2P_LISTEN_PORT
func (c Config) OCR2P2PListenPortRaw() string {
	return c.viper.GetString(EnvVarName("OCR2P2PListenPort"))
}

// OCR2P2PAnnounceIP is an optional override. If specified it will force the p2p
// layer to announce this IP as the externally reachable one to the DHT
// If this is set, OCR2P2PAnnouncePort MUST also be set.
func (c Config) OCR2P2PAnnounceIP() net.IP {
	str := c.viper.GetString(EnvVarName("OCR2P2PAnnounceIP"))
	return net.ParseIP(str)
}

// OCR2P2PAnnouncePort is an optional override. If specified it will force the p2p
// layer to announce this port as the externally reachable one to the DHT.
// If this is set, OCR2P2PAnnounceIP MUST also be set.
func (c Config) OCR2P2PAnnouncePort() uint16 {
	return uint16(c.viper.GetUint32(EnvVarName("OCR2P2PAnnouncePort")))
}

// OCR2P2PDHTAnnouncementCounterUserPrefix can be used to restore the node's
// ability to announce its IP/port on the OCR2P2P network after a database
// rollback. Make sure to only increase this value, and *never* decrease it.
// Don't use this variable unless you really know what you're doing, since you
// could semi-permanently exclude your node from the OCR2P2P network by
// misconfiguring it.
func (c Config) OCR2P2PDHTAnnouncementCounterUserPrefix() uint32 {
	return c.viper.GetUint32(EnvVarName("OCR2P2PDHTAnnouncementCounterUserPrefix"))
}

func (c Config) OCR2P2PPeerstoreWriteInterval() time.Duration {
	return c.getWithFallback("OCR2P2PPeerstoreWriteInterval", parseDuration).(time.Duration)
}

// OCR2P2PPeerID is the default peer ID that will be used, if not overridden
func (c Config) OCR2P2PPeerID(override *p2pkey.PeerID) (p2pkey.PeerID, error) {
	if override != nil {
		return *override, nil
	}
	pidStr := c.viper.GetString(EnvVarName("OCR2P2PPeerID"))
	if pidStr != "" {
		var pid p2pkey.PeerID
		err := pid.UnmarshalText([]byte(pidStr))
		if err != nil {
			return "", errors.Wrapf(ErrInvalid, "OCR2_P2P_PEER_ID is invalid %v", err)
		}
		return pid, nil
	}
	return "", errors.Wrap(ErrUnset, "OCR2_P2P_PEER_ID")
}

func (c Config) OCR2P2PPeerIDIsSet() bool {
	return c.viper.GetString(EnvVarName("OCR2P2PPeerID")) != ""
}

// OCR2P2PPeerIDRaw returns the string value of whatever OCR2_P2P_PEER_ID was set to with no parsing
func (c Config) OCR2P2PPeerIDRaw() string {
	return c.viper.GetString(EnvVarName("OCR2P2PPeerID"))
}

func (c Config) OCR2P2PBootstrapPeers(override []string) ([]string, error) {
	if override != nil {
		return override, nil
	}
	if c.viper.IsSet(EnvVarName("OCR2P2PBootstrapPeers")) {
		bps := c.viper.GetStringSlice(EnvVarName("OCR2P2PBootstrapPeers"))
		if bps != nil {
			return bps, nil
		}
		return nil, errors.Wrap(ErrUnset, "OCR2_P2P_BOOTSTRAP_PEERS")
	}
	return []string{}, nil
}

// OCR2P2PNetworkingStack returns the preferred networking stack for libocr
func (c Config) OCR2P2PNetworkingStack() (n ocrnetworking.NetworkingStack) {
	str := c.OCR2P2PNetworkingStackRaw()
	err := n.UnmarshalText([]byte(str))
	if err != nil {
		logger.Fatalf("OCR2P2PNetworkingStack failed to unmarshal '%s': %s", str, err)
	}
	return n
}

// OCR2P2PNetworkingStackRaw returns the raw string passed as networking stack
func (c Config) OCR2P2PNetworkingStackRaw() string {
	return c.viper.GetString(EnvVarName("OCR2P2PNetworkingStack"))
}

// OCR2P2PV2ListenAddresses contains the addresses the peer will listen to on the network in <host>:<port> form as
// accepted by net.Listen, but host and port must be fully specified and cannot be empty.
func (c Config) OCR2P2PV2ListenAddresses() []string {
	return c.viper.GetStringSlice(EnvVarName("OCR2P2PV2ListenAddresses"))
}

// OCR2P2PV2AnnounceAddresses contains the addresses the peer will advertise on the network in <host>:<port> form as
// accepted by net.Dial. The addresses should be reachable by peers of interest.
func (c Config) OCR2P2PV2AnnounceAddresses() []string {
	if c.viper.IsSet(EnvVarName("OCR2P2PV2AnnounceAddresses")) {
		return c.viper.GetStringSlice(EnvVarName("OCR2P2PV2AnnounceAddresses"))
	}
	return c.OCR2P2PV2ListenAddresses()
}

// OCR2P2PV2AnnounceAddressesRaw returns the raw value passed in
func (c Config) OCR2P2PV2AnnounceAddressesRaw() []string {
	return c.viper.GetStringSlice(EnvVarName("OCR2P2PV2AnnounceAddresses"))
}

// OCR2P2PV2Bootstrappers returns the default bootstrapper peers for libocr's v2
// networking stack
func (c Config) OCR2P2PV2Bootstrappers() (locators []ocrcommontypes.BootstrapperLocator) {
	bootstrappers := c.OCR2P2PV2BootstrappersRaw()
	for _, s := range bootstrappers {
		var locator ocrcommontypes.BootstrapperLocator
		err := locator.UnmarshalText([]byte(s))
		if err != nil {
			logger.Fatalf("invalid format for bootstrapper '%s', got error: %s", s, err)
		}
		locators = append(locators, locator)
	}
	return
}

// OCR2P2PV2BootstrappersRaw returns the raw strings for v2 bootstrap peers
func (c Config) OCR2P2PV2BootstrappersRaw() []string {
	return c.viper.GetStringSlice(EnvVarName("OCR2P2PV2Bootstrappers"))
}

// OCR2P2PV2DeltaDial controls how far apart Dial attempts are
func (c Config) OCR2P2PV2DeltaDial() models.Duration {
	return models.MustMakeDuration(c.getWithFallback("OCR2P2PV2DeltaDial", parseDuration).(time.Duration))
}

// OCR2P2PV2DeltaReconcile controls how often a Reconcile message is sent to every peer.
func (c Config) OCR2P2PV2DeltaReconcile() models.Duration {
	return models.MustMakeDuration(c.getWithFallback("OCR2P2PV2DeltaReconcile", parseDuration).(time.Duration))
}

func (c Config) OCR2BootstrapCheckInterval() time.Duration {
	return c.getWithFallback("OCR2BootstrapCheckInterval", parseDuration).(time.Duration)
}

func (c Config) OCR2ContractTransmitterTransmitTimeout() time.Duration {
	return c.getWithFallback("OCR2ContractTransmitterTransmitTimeout", parseDuration).(time.Duration)
}

func (c Config) OCR2ObservationTimeout(override time.Duration) time.Duration {
	return c.getDurationWithOverride(override, "OCR2ObservationTimeout")
}

func (c Config) OCR2ObservationGracePeriod() time.Duration {
	return c.getWithFallback("OCR2ObservationGracePeriod", parseDuration).(time.Duration)
}

func (c Config) OCR2BlockchainTimeout(override time.Duration) time.Duration {
	return c.getDurationWithOverride(override, "OCR2BlockchainTimeout")
}

func (c Config) OCR2ContractSubscribeInterval(override time.Duration) time.Duration {
	return c.getDurationWithOverride(override, "OCR2ContractSubscribeInterval")
}

func (c Config) OCR2ContractPollInterval(override time.Duration) time.Duration {
	return c.getDurationWithOverride(override, "OCR2ContractPollInterval")
}

func (c Config) OCR2ContractConfirmations(override uint16) uint16 {
	if override != uint16(0) {
		return override
	}
	if c.viper.IsSet(EnvVarName("OCR2ContractConfirmations")) {
		return uint16(c.viper.GetUint32(EnvVarName("OCR2ContractConfirmations")))
	}
	return chainSpecificConfig(c).OCR2ContractConfirmations
}

func (c Config) OCR2DatabaseTimeout() time.Duration {
	return c.getWithFallback("OCR2DatabaseTimeout", parseDuration).(time.Duration)
}

func (c Config) OCR2DHTLookupInterval() int {
	return int(c.getWithFallback("OCR2DHTLookupInterval", parseUint16).(uint16))
}

func (c Config) OCR2IncomingMessageBufferSize() int {
	return int(c.getWithFallback("OCR2IncomingMessageBufferSize", parseUint16).(uint16))
}

func (c Config) OCR2NewStreamTimeout() time.Duration {
	return c.getWithFallback("OCR2NewStreamTimeout", parseDuration).(time.Duration)
}

func (c Config) OCR2OutgoingMessageBufferSize() int {
	return int(c.getWithFallback("OCR2IncomingMessageBufferSize", parseUint16).(uint16))
}

// OCR2TraceLogging determines whether OCR2 logs at TRACE level are enabled. The
// option to turn them off is given because they can be very verbose
func (c Config) OCR2TraceLogging() bool {
	return c.viper.GetBool(EnvVarName("OCR2TraceLogging"))
}

func (c Config) OCR2MonitoringEndpoint(override string) string {
	if override != "" {
		return override
	}
	return c.viper.GetString(EnvVarName("OCR2MonitoringEndpoint"))
}

// OCR2DefaultTransactionQueueDepth controls the queue size for DropOldestStrategy in OCR2
// Set to 0 to use SendEvery strategy instead
func (c Config) OCR2DefaultTransactionQueueDepth() uint32 {
	return c.viper.GetUint32(EnvVarName("OCR2DefaultTransactionQueueDepth"))
}

func (c Config) OCR2TransmitterAddress(override *ethkey.EIP55Address) (ethkey.EIP55Address, error) {
	if override != nil {
		return *override, nil
	}
	taStr := c.viper.GetString(EnvVarName("OCR2TransmitterAddress"))
	if taStr != "" {
		ta, err := ethkey.NewEIP55Address(taStr)
		if err != nil {
			return "", errors.Wrapf(ErrInvalid, "OCR2_TRANSMITTER_ADDRESS is invalid EIP55 %v", err)
		}
		return ta, nil
	}
	return "", errors.Wrap(ErrUnset, "OCR2_TRANSMITTER_ADDRESS")
}

func (c Config) OCR2KeyBundleID(override *models.Sha256Hash) (models.Sha256Hash, error) {
	if override != nil {
		return *override, nil
	}
	kbStr := c.viper.GetString(EnvVarName("OCR2KeyBundleID"))
	if kbStr != "" {
		kb, err := models.Sha256HashFromHex(kbStr)
		if err != nil {
			return models.Sha256Hash{}, errors.Wrapf(ErrInvalid, "OCR2_KEY_BUNDLE_ID is an invalid sha256 hash hex string %v", err)
		}
		return kb, nil
	}
	return models.Sha256Hash{}, errors.Wrap(ErrUnset, "OCR2_KEY_BUNDLE_ID")
}
