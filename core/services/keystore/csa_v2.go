package keystore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/csakey"
)

//go:generate mockery --name CSA --output mocks/ --case=underscore

var ErrCSAKeyExists = errors.New("CSA key does not exist")

// type CSAKeystoreInterface interface {
type CSA interface {
	CreateCSAKey() (*csakey.KeyV2, error)
	CountCSAKeys() (int64, error)
	ListCSAKeys() ([]csakey.KeyV2, error)
	ImportKey([]byte, string) (csakey.KeyV2, error)
	ExportKey(string, string) ([]byte, error)
	GetV1KeysAsV2() ([]csakey.KeyV2, error)
}

type csa struct {
	*keyManager
}

var _ CSA = csa{}

func newCSAKeyStore(km *keyManager) csa {
	return csa{
		km,
	}
}

func (ks csa) CreateCSAKey() (*csakey.KeyV2, error) {
	ks.lock.Lock()
	defer ks.lock.Unlock()
	if ks.isLocked() {
		return nil, LockedErr
	}
	// Ensure you can only have one CSA at a time. This is a temporary
	// restriction until we are able to handle multiple CSA keys in the
	// communication channel
	if len(ks.keyRing.CSA) > 0 {
		return nil, errors.New("can only have 1 CSA key")
	}
	key, err := csakey.NewV2()
	if err != nil {
		return nil, err
	}
	return &key, ks.safeAddKey(key)
}

func (ks csa) CountCSAKeys() (int64, error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return 0, LockedErr
	}
	return int64(len(ks.keyRing.CSA)), nil
}

func (ks csa) ListCSAKeys() (keys []csakey.KeyV2, _ error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return nil, LockedErr
	}
	for _, key := range ks.keyRing.CSA {
		keys = append(keys, key)
	}
	return keys, nil
}

func (ks csa) ExportKey(keyID string, password string) ([]byte, error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return nil, LockedErr
	}
	key, err := ks.getCSAKey(keyID)
	if err != nil {
		return nil, err
	}
	return key.ToEncryptedJSON(password, ks.scryptParams)
}

func (ks csa) ImportKey(keyJSON []byte, password string) (csakey.KeyV2, error) {
	ks.lock.Lock()
	defer ks.lock.Unlock()
	if ks.isLocked() {
		return csakey.KeyV2{}, LockedErr
	}
	key, err := csakey.FromEncryptedJSON(keyJSON, password)
	if err != nil {
		return key, errors.Wrap(err, "EthKeyStore#ImportKey failed to decrypt key")
	}
	return key, ks.keyManager.safeAddKey(key)
}

// ListCSAKeys lists all CSA keys.
func (ks csa) GetV1KeysAsV2() (keys []csakey.KeyV2, _ error) {
	v1Keys, err := ks.GetEncryptedV1CSAKeys(context.Background())
	if err != nil {
		return keys, err
	}
	for _, keyV1 := range v1Keys {
		err := keyV1.Unlock(ks.password)
		if err != nil {
			return keys, err
		}
		keys = append(keys, keyV1.ToV2())
	}
	return keys, nil
}

func (ks csa) getCSAKey(keyID string) (csakey.KeyV2, error) {
	key, found := ks.keyManager.keyRing.CSA[keyID]
	if found {
		return key, nil
	}
	v1Keys, err := ks.GetV1KeysAsV2()
	if err != nil {
		return csakey.KeyV2{}, err
	}
	for _, key := range v1Keys {
		if key.ID() == keyID {
			return key, nil
		}
	}
	return csakey.KeyV2{}, errors.New("not found")
}
