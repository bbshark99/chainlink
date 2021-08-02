package keystore

import (
	"fmt"
	"math/big"

	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/vrfkey"
	"github.com/smartcontractkit/chainlink/core/services/signatures/secp256k1"
	"github.com/smartcontractkit/chainlink/core/utils"
)

// ErrMatchingVRFKey is returned when Import attempts to import key with a
// PublicKey matching one already in the database
var ErrMatchingVRFKey = errors.New(
	`key with matching public key already stored in DB`)

// ErrAttemptToDeleteNonExistentKeyFromDB is returned when Delete is asked to
// delete a key it can't find in the DB.
var ErrAttemptToDeleteNonExistentKeyFromDB = errors.New("key is not present in DB")

type VRF interface {
	GenerateProof(id string, seed *big.Int) (vrfkey.Proof, error)
	CreateKey() (*vrfkey.KeyV2, error)
	Store(key *vrfkey.KeyV2, phrase string, scryptParams utils.ScryptParams) error
	StoreInMemoryXXXTestingOnly(key *vrfkey.KeyV2)
	Delete(id string) (err error)
	Import(keyjson []byte, auth string) (*vrfkey.KeyV2, error)
	Export(id string, password string) ([]byte, error)
	Get(id string) (*vrfkey.KeyV2, error)
	GetAll() ([]vrfkey.KeyV2, error)
}

type vrf struct {
	*keyManager
}

var _ VRF = vrf{}

func newVRFKeyStore(km *keyManager) vrf {
	return vrf{
		km,
	}
}

func (ks vrf) GenerateProof(id string, seed *big.Int) (vrfkey.Proof, error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return vrfkey.Proof{}, LockedErr
	}
	key, err := ks.getVRFKey(id)
	if err != nil {
		return vrfkey.Proof{}, err
	}
	return key.GenerateProof(seed)
}

func (ks vrf) CreateKey() (*vrfkey.KeyV2, error) {
	ks.lock.Lock()
	defer ks.lock.Unlock()
	if ks.isLocked() {
		return nil, LockedErr
	}
	key, err := vrfkey.NewV2()
	if err != nil {
		return nil, errors.Wrapf(err, "while generating new vrf key")
	}
	err = ks.safeAddKey(key)
	if err != nil {
		return nil, errors.Wrapf(err, "while adding new vrf key")
	}
	return &key, nil
}

func (ks vrf) CreateAndUnlockWeakInMemoryEncryptedKeyXXXTestingOnly(phrase string) (*vrfkey.KeyV2, error) {
	return nil, nil
}

func (ks vrf) Store(key *vrfkey.KeyV2, phrase string, scryptParams utils.ScryptParams) error {
	return nil
}

func (ks vrf) StoreInMemoryXXXTestingOnly(key *vrfkey.KeyV2) {
	ks.keyRing.VRF[key.ID()] = *key
}

func (ks vrf) Delete(id string) (err error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return LockedErr
	}
	key, err := ks.Get(id)
	if err != nil {
		return err
	}
	return ks.safeRemoveKey(key)
	// TODO - RYAN
	// return ks.safeRemoveKey("VRF", id)
}

func (ks vrf) Import(keyJSON []byte, password string) (*vrfkey.KeyV2, error) {
	ks.lock.Lock()
	defer ks.lock.Unlock()
	if ks.isLocked() {
		return nil, LockedErr
	}
	key, err := vrfkey.FromEncryptedJSON(keyJSON, password)
	if err != nil {
		return nil, errors.Wrap(err, "VRFKeyStore#ImportKey failed to decrypt key")
	}
	if _, found := ks.keyRing.VRF[key.ID()]; found {
		return nil, errors.New("VRFKeyStore#ImportKey key already exists")
	}
	return &key, ks.safeAddKey(key)
}

func (ks vrf) Export(id string, password string) ([]byte, error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return nil, LockedErr
	}
	key, err := ks.Get(id)
	if err != nil {
		return nil, err
	}
	return key.ToEncryptedJSON(password, ks.scryptParams)
}

func (ks vrf) Get(id string) (*vrfkey.KeyV2, error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return nil, LockedErr
	}
	key, found := ks.keyRing.VRF[id]
	if !found {
		return nil, errors.New(fmt.Sprintf("VRF key not found with ID %s", id))
	}
	return &key, nil
}

func (ks vrf) GetAll() (keys []vrfkey.KeyV2, _ error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return keys, LockedErr
	}
	for _, key := range ks.keyRing.VRF {
		keys = append(keys, key)
	}
	return keys, nil
}

func (ks vrf) GetSpecificKey(k secp256k1.PublicKey) (*vrfkey.KeyV2, error) {
	return nil, nil
}

func (ks vrf) getVRFKey(id string) (vrfkey.KeyV2, error) {
	key, found := ks.keyRing.VRF[id]
	if !found {
		return vrfkey.KeyV2{}, errors.New(fmt.Sprintf("VRF key not found with ID %s", id))
	}
	return key, nil
}
