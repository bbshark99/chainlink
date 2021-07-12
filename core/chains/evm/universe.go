package evm

import (
	"context"

	"github.com/smartcontractkit/chainlink/core/logger"
	"github.com/smartcontractkit/chainlink/core/service"
	"github.com/smartcontractkit/chainlink/core/services"
	"github.com/smartcontractkit/chainlink/core/services/bulletprooftxmanager"
	"github.com/smartcontractkit/chainlink/core/services/eth"
	"github.com/smartcontractkit/chainlink/core/services/headtracker"
	httypes "github.com/smartcontractkit/chainlink/core/services/headtracker/types"
	"github.com/smartcontractkit/chainlink/core/services/keystore"
	"github.com/smartcontractkit/chainlink/core/services/log"
	"github.com/smartcontractkit/chainlink/core/services/postgres"
	"github.com/smartcontractkit/chainlink/core/utils"
	"go.uber.org/multierr"
	"gorm.io/gorm"
)

func LoadUniverses(globalLogger *logger.Logger, db *gorm.DB, config UniverseConfig, keyStore bulletprooftxmanager.KeyStore, advisoryLocker postgres.AdvisoryLocker, eventBroadcaster postgres.EventBroadcaster) (universes []Universe, err error) {
	var chains []EVMChain
	err = db.Preload("Nodes").Find(&chains).Error
	if err != nil {
		return nil, err
	}
	for _, chain := range chains {
		universe, err2 := NewUniverse(chain, globalLogger, db, config, keyStore, advisoryLocker, eventBroadcaster)
		err = multierr.Combine(err, err2)
		universes = append(universes, universe)
	}
	return universes, err
}

// Universe is an eth-compatible chain along with every accompanying service
type Universe interface {
	service.Service
	// TODO: Can we reduce surface area by directly exposing methods?
	GetHeadBroadcaster() httypes.HeadBroadcaster
	GetHeadTracker() httypes.Tracker
	// GetTxManager() bulletprooftxmanager.TxManager
	// GetChain() Chain
}

// TODO: Reconcile with chain-specific config
type UniverseConfig interface {
	bulletprooftxmanager.Config
	headtracker.Config
	log.Config
	BalanceMonitorEnabled() bool
}

type concreteUniverse struct {
	utils.StartStopOnce
	chain           EVMChain
	client          eth.Client
	txm             bulletprooftxmanager.TxManager
	logger          *logger.Logger
	headBroadcaster httypes.HeadBroadcaster
	headTracker     httypes.Tracker
	logBroadcaster  log.Broadcaster
	balanceMonitor  services.BalanceMonitor
}

func NewUniverse(chain EVMChain, globalLogger *logger.Logger, db *gorm.DB, config UniverseConfig, keyStore *keystore.Eth, advisoryLocker postgres.AdvisoryLocker, eventBroadcaster postgres.EventBroadcaster) (Universe, error) {
	// l := globalLogger.With("chainID", chain.ID.String())
	ethClient, err := NewEthClientFromChain(chain)
	if err != nil {
		return nil, err
	}
	serviceLogLevels, err := globalLogger.GetServiceLogLevels()
	if err != nil {
		return nil, err
	}
	headTrackerLogger, err := globalLogger.InitServiceLevelLogger(logger.HeadTracker, serviceLogLevels[logger.HeadTracker])
	if err != nil {
		return nil, err
	}
	headBroadcaster := headtracker.NewHeadBroadcaster()
	orm := headtracker.NewORM(db, *chain.ID.ToInt())
	headTracker := headtracker.NewHeadTracker(headTrackerLogger, ethClient, config, orm, headBroadcaster)
	txm := bulletprooftxmanager.NewBulletproofTxManager(db, ethClient, config, keyStore, advisoryLocker, eventBroadcaster)

	// Highest seen head height is used as part of the start of LogBroadcaster backfill range
	highestSeenHead, err2 := headTracker.HighestSeenHeadFromDB()
	if err2 != nil {
		return nil, err2
	}

	var balanceMonitor services.BalanceMonitor
	if config.BalanceMonitorEnabled() {
		balanceMonitor = services.NewBalanceMonitor(db, ethClient, keyStore)
	}

	logBroadcaster := log.NewBroadcaster(log.NewORM(db), ethClient, config, highestSeenHead)
	u := concreteUniverse{
		chain:           chain,
		txm:             txm,
		headBroadcaster: headBroadcaster,
		headTracker:     headTracker,
		logBroadcaster:  logBroadcaster,
		balanceMonitor:  balanceMonitor,
	}
	return &u, nil
}

func (u *concreteUniverse) Start() error {
	return u.StartOnce("Universe", func() (merr error) {
		// EthClient must be dialed first because subsequent services may make eth
		// calls on startup
		merr = multierr.Combine(
			u.client.Dial(context.TODO()),
			u.txm.Start(),
			u.headBroadcaster.Start(),
			u.headTracker.Start(),
			u.logBroadcaster.Start(),
		)
		if u.balanceMonitor != nil {
			merr = multierr.Combine(merr, u.balanceMonitor.Start())
		}
		return merr
	})
}

func (u *concreteUniverse) Close() error {
	return u.StopOnce("Universe", func() (merr error) {
		if u.balanceMonitor != nil {
			merr = u.balanceMonitor.Close()
		}
		merr = multierr.Combine(
			u.logBroadcaster.Close(),
			u.headTracker.Stop(),
			u.headBroadcaster.Close(),
			u.txm.Close(),
		)
		u.client.Close()
		return merr
	})
}

func (u *concreteUniverse) GetHeadBroadcaster() httypes.HeadBroadcaster {
	return u.headBroadcaster
}

func (u *concreteUniverse) GetHeadTracker() httypes.Tracker {
	return u.headTracker
}
