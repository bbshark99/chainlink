package keystore

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/ethkey"
	"gorm.io/gorm"
)

//go:generate mockery --name Eth --output mocks/ --case=underscore

// Eth is the external interface for EthKeyStore
type Eth interface {
	// Requires Unlock
	CreateNewKey() (ethkey.KeyV2, error)
	EnsureFundingKey() (key ethkey.KeyV2, didExist bool, err error)
	ImportKey(keyJSON []byte, oldPassword string) (ethkey.KeyV2, error)
	ExportKey(address common.Address, newPassword string) ([]byte, error)
	AddKey(key ethkey.KeyV2) error
	RemoveKey(address common.Address, hardDelete bool) (deletedKey ethkey.KeyV2, err error)
	SubscribeToKeyChanges() (ch chan struct{}, unsub func())

	SignTx(fromAddress common.Address, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)

	AllKeys() (keys []ethkey.KeyV2, err error)
	SendingKeys() (keys []ethkey.KeyV2, err error)
	FundingKeys() (keys []ethkey.KeyV2, err error)
	IsFundingKey(key ethkey.KeyV2) (bool, error)
	KeyByAddress(address common.Address) (ethkey.KeyV2, error)
	HasSendingKeyWithAddress(address common.Address) (bool, error)
	GetRoundRobinAddress(addresses ...common.Address) (address common.Address, err error)

	GetStateForAddress(common.Address) (ethkey.State, error) // TODO - RYAN - GetState()
	GetStateForKey(ethkey.KeyV2) (ethkey.State, error)
	GetStatesForKeys([]ethkey.KeyV2) ([]ethkey.State, error)

	// Does not require Unlock
	HasDBSendingKeys() (bool, error)
	ImportKeyFileToDB(keyPath string) (ethkey.KeyV2, error)
}

type eth struct {
	*keyManager
}

var _ Eth = eth{}

func newEthKeyStore(km *keyManager) eth {
	return eth{
		km,
	}
}

// Requires Unlock
func (ks eth) CreateNewKey() (key ethkey.KeyV2, _ error) {
	ks.lock.Lock()
	defer ks.lock.Unlock()
	if ks.isLocked() {
		return key, LockedErr
	}
	key, err := ethkey.NewV2()
	if err != nil {
		return key, err
	}
	return key, ks.addEthKey(key)
}

func (ks eth) EnsureFundingKey() (ethkey.KeyV2, bool, error) {
	ks.lock.Lock()
	defer ks.lock.Unlock()
	if ks.isLocked() {
		return ethkey.KeyV2{}, false, LockedErr
	}
	fundingKeys, err := ks.fundingKeys()
	if err != nil {
		return ethkey.KeyV2{}, false, LockedErr
	}
	if len(fundingKeys) > 0 {
		return fundingKeys[0], true, nil
	}
	key, err := ethkey.NewV2()
	if err != nil {
		return ethkey.KeyV2{}, false, err
	}
	return key, false, ks.addEthKeyWithState(key, ethkey.State{IsFunding: true})
}

func (ks eth) ImportKey(keyJSON []byte, password string) (key ethkey.KeyV2, _ error) {
	ks.lock.Lock()
	defer ks.lock.Unlock()
	if ks.isLocked() {
		return ethkey.KeyV2{}, LockedErr
	}
	key, err := ethkey.FromEncryptedJSON(keyJSON, password)
	if err != nil {
		return key, errors.Wrap(err, "EthKeyStore#ImportKey failed to decrypt key")
	}
	return key, ks.addEthKey(key)
}

func (ks eth) ExportKey(address common.Address, password string) ([]byte, error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return nil, LockedErr
	}
	key, err := ks.getEthKey(ethkey.EIP55AddressFromAddress(address))
	if err != nil {
		return nil, err
	}
	return key.ToEncryptedJSON(password, ks.scryptParams)
}

// TODO - change argument to keyV2 once v2 migration is complete
func (ks eth) AddKey(key ethkey.KeyV2) error {
	ks.lock.Lock()
	defer ks.lock.Unlock()
	if ks.isLocked() {
		return LockedErr
	}
	return ks.addEthKey(key)
}

// TODO - remove "soft delete" option once V2 migration is complete
func (ks eth) RemoveKey(address common.Address, hardDelete bool) (ethkey.KeyV2, error) {
	ks.lock.Lock()
	defer ks.lock.Unlock()
	if ks.isLocked() {
		return ethkey.KeyV2{}, LockedErr
	}
	if !hardDelete {
		return ethkey.KeyV2{}, errors.New("soft delete not available")
	}
	key, err := ks.getEthKey(ethkey.EIP55AddressFromAddress(address))
	if err != nil {
		return ethkey.KeyV2{}, err
	}
	err = ks.safeRemoveKey(key, func(db *gorm.DB) error {
		return db.Where("address = ?", key.Address).Delete(ethkey.State{}).Error
	})
	return key, err
}

func (ks eth) SubscribeToKeyChanges() (ch chan struct{}, unsub func()) {
	return nil, func() {}
}

func (ks eth) SignTx(address common.Address, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return nil, LockedErr
	}
	key, err := ks.getEthKey(ethkey.EIP55AddressFromAddress(address))
	if err != nil {
		return nil, err
	}
	signer := types.LatestSignerForChainID(chainID)
	return types.SignTx(tx, signer, key.ToEcdsaPrivKey())
}

func (ks eth) AllKeys() (keys []ethkey.KeyV2, _ error) {
	ks.lock.Lock()
	defer ks.lock.Unlock()
	if ks.isLocked() {
		return nil, LockedErr
	}
	for _, key := range ks.keyRing.Eth {
		keys = append(keys, key)
	}
	return keys, nil
}

func (ks eth) SendingKeys() (sendingKeys []ethkey.KeyV2, err error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return nil, LockedErr
	}
	states, err := ks.getEthKeyStatesWhere("is_funding = ?", false)
	if err != nil {
		return sendingKeys, err
	}
	for _, state := range states {
		sendingKeys = append(sendingKeys, ks.keyRing.Eth[state.KeyID()])
	}
	return sendingKeys, nil
}

func (ks eth) FundingKeys() (fundingKeys []ethkey.KeyV2, err error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return nil, LockedErr
	}
	return ks.fundingKeys()
}

func (ks eth) IsFundingKey(key ethkey.KeyV2) (bool, error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return false, LockedErr
	}
	_, err := ks.getEthKeyStateWhere("is_funding = ? AND address = ?", true, key.Address)
	if err == gorm.ErrRecordNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (ks eth) KeyByAddress(address common.Address) (ethkey.KeyV2, error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return ethkey.KeyV2{}, LockedErr
	}
	return ks.getEthKey(ethkey.EIP55AddressFromAddress(address))
}

func (ks eth) HasSendingKeyWithAddress(address common.Address) (bool, error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return false, LockedErr
	}
	_, err := ks.getEthKeyStateWhere("is_funding = ? AND address = ?", false, address)
	if err == gorm.ErrRecordNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (ks eth) GetRoundRobinAddress(whitelist ...common.Address) (common.Address, error) {
	// TODO - RYAN - does this actually need any locks?
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return common.Address{}, LockedErr
	}
	state, err := ks.getNextRoundRobinAddress(whitelist)
	if err != nil {
		return common.Address{}, err
	}
	if state.Address.IsZeroAddress() {
		return common.Address{}, errors.New("Could not find round round robin address")
	}
	return state.Address.Address(), nil
}

func (ks eth) GetStateForAddress(address common.Address) (ethkey.State, error) {
	return ks.getEthKeyStateWhere("address = ?", address)
}

func (ks eth) GetStateForKey(key ethkey.KeyV2) (ethkey.State, error) {
	return ks.getEthKeyStateWhere("address = ?", key.Address)
}

func (ks eth) GetStatesForKeys(keys []ethkey.KeyV2) ([]ethkey.State, error) {
	var addresses []ethkey.EIP55Address
	for _, key := range keys {
		addresses = append(addresses, key.Address)
	}
	return ks.getEthKeyStatesWhere("address in ?", addresses)
}

// Does not require Unlock
func (ks eth) HasDBSendingKeys() (bool, error) {
	_, err := ks.getEthKeyStateWhere("is_funding = ?", false)
	if err == gorm.ErrRecordNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (ks eth) ImportKeyFileToDB(keyPath string) (ethkey.KeyV2, error) {
	return ethkey.KeyV2{}, errors.New("deprecated: use remote client!")
}

func (ks eth) getEthKey(address ethkey.EIP55Address) (ethkey.KeyV2, error) {
	key := ks.keyRing.Eth[address.Hex()]
	if key.Address.IsZeroAddress() {
		return ethkey.KeyV2{}, newNoKeyError(address)
	}
	return key, nil
}

func (ks eth) fundingKeys() (fundingKeys []ethkey.KeyV2, err error) {
	states, err := ks.getEthKeyStatesWhere("is_funding = ?", true)
	if err != nil {
		return fundingKeys, err
	}
	for _, state := range states {
		fundingKeys = append(fundingKeys, ks.keyRing.Eth[state.KeyID()])
	}
	return fundingKeys, nil
}

// caller must hold lock!
func (ks eth) addEthKey(key ethkey.KeyV2) error {
	return ks.addEthKeyWithState(key, ethkey.State{})
}

func (ks eth) addEthKeyWithState(key ethkey.KeyV2, state ethkey.State) error {
	state.Address = key.Address
	return ks.safeAddKey(key, func(db *gorm.DB) error {
		return db.Create(&state).Error
	})
}

func newNoKeyError(address ethkey.EIP55Address) error {
	return errors.Errorf("address %s not in keystore", address.Hex())
}
