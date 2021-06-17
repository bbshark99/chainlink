package keystore

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/csakey"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/ethkey"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/ocrkey"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/p2pkey"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/vrfkey"
	"github.com/smartcontractkit/chainlink/core/services/postgres"
	"github.com/smartcontractkit/chainlink/core/utils"
	"gorm.io/gorm"
)

var LockedErr = errors.New("Keystore is locked")

type Master interface {
	CSA() CSA
	Eth() Eth
	OCR() OCR
	VRF() VRF
	Unlock(password string) error
	Migrate() error
}

type masterV2 struct {
	*keyManager
	csa csa
	eth eth
	ocr ocr
	vrf vrf
}

func New(db *gorm.DB, scryptParams utils.ScryptParams) Master {
	return newV2(db, scryptParams)
}

// TODO - combine these functions
func newV2(db *gorm.DB, scryptParams utils.ScryptParams) *masterV2 {
	km := &keyManager{
		ksORM:        NewORM(db),
		scryptParams: scryptParams,
		lock:         &sync.RWMutex{},
	}

	return &masterV2{
		keyManager: km,
		csa:        newCSAKeyStore(km),
		eth:        newEthKeyStore(km),
		ocr:        newOCRKeyStore(km),
		vrf:        newVRFKeyStore(km),
	}
}

func (ks masterV2) CSA() CSA {
	return ks.csa
}

func (ks *masterV2) Eth() Eth {
	return ks.eth
}

func (ks *masterV2) OCR() OCR {
	return ks.ocr
}

func (ks *masterV2) VRF() VRF {
	return ks.vrf
}

func (ks *masterV2) Migrate() error {
	ks.lock.Lock()
	defer ks.lock.Unlock()
	if ks.isLocked() {
		return LockedErr
	}
	csaKeys, err := ks.csa.GetV1KeysAsV2()
	if err != nil {
		return err
	}
	for _, csakey := range csaKeys {
		ks.keyRing.CSA[csakey.ID()] = csakey
	}
	// ocrKeys, err := ks.ocr.GetV1KeysAsV2()
	// if err != nil {
	// 	return err
	// }
	// for _, ocrkey := range ocrKeys {
	// 	ks.keyRing.OCR[ocrkey.ID()] = ocrkey
	// }
	// vrfKeys, err := ks.vrf.GetV1KeysAsV2()
	// if err != nil {
	// 	return err
	// }
	// for _, vrfkey := range vrfKeys {
	// 	ks.keyRing.VRF[vrfkey.ID()] = vrfkey
	// }
	if err = ks.keyManager.save(); err != nil {
		return err
	}
	// ethKeys, err := ks.eth.GetV1KeysAsV2()
	// if err != nil {
	// 	return err
	// }
	// for _, ethkey := range ethKeys {
	// 	if err = ks.eth.addEthKey(ethkey); err != nil {
	// 		return err
	// 	}
	// }
	return nil
}

type keyManager struct {
	ksORM
	scryptParams utils.ScryptParams
	keyRing      *KeyRing // TODO - RYAN - pointer?
	lock         *sync.RWMutex
	password     string
}

func (km *keyManager) Unlock(password string) error {
	km.lock.Lock()
	defer km.lock.Unlock()
	if km.password != "" {
		return errors.New("keystore already unlocked")
	}
	ekr, err := km.getEncryptedKeyRing()
	if err != nil {
		return errors.Wrap(err, "unable to get encrypted key ring")
	}
	kr, err := ekr.Decrypt(password)
	if err != nil {
		return errors.Wrap(err, "unable to decrypt encrypted key ring")
	}
	km.keyRing = &kr
	km.password = password
	return nil
}

// lock needs to be held by caller!!!
func (km *keyManager) save(callbacks ...func(*gorm.DB) error) error {
	ekb, err := km.keyRing.Encrypt(km.password, km.scryptParams)
	if err != nil {
		return errors.Wrap(err, "unable to encrypt keyRing")
	}
	return postgres.GormTransactionWithDefaultContext(km.db, func(tx *gorm.DB) error {
		err := km.withDB(tx).saveEncryptedKeyRing(&ekb)
		if err != nil {
			return err
		}
		for _, callback := range callbacks {
			err = callback(tx)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// caller must hold lock!
func (km *keyManager) safeAddKey(unknownKey Key, callbacks ...func(*gorm.DB) error) error {
	fieldName, err := getFieldNameForKey(unknownKey)
	if err != nil {
		return err
	}
	// add key to keyring
	id := reflect.ValueOf(unknownKey.ID())
	key := reflect.ValueOf(unknownKey)
	keyRing := reflect.Indirect(reflect.ValueOf(km.keyRing))
	keyMap := keyRing.FieldByName(fieldName)
	keyMap.SetMapIndex(id, key)
	// save keyring to DB
	err = km.save(callbacks...)
	// if save fails, remove key from keyring
	if err != nil {
		keyMap.SetMapIndex(id, reflect.Value{})
		return err
	}
	return nil
}

// caller must hold lock!
func (km *keyManager) safeRemoveKey(unknownKey Key, callbacks ...func(*gorm.DB) error) (err error) {
	fieldName, err := getFieldNameForKey(unknownKey)
	if err != nil {
		return err
	}
	id := reflect.ValueOf(unknownKey.ID())
	key := reflect.ValueOf(unknownKey)
	keyRing := reflect.Indirect(reflect.ValueOf(km.keyRing))
	keyMap := keyRing.FieldByName(fieldName)
	keyMap.SetMapIndex(id, reflect.Value{})
	// save keyring to DB
	err = km.save(callbacks...)
	// if save fails, add key back to keyRing
	if err != nil {
		keyMap.SetMapIndex(id, key)
		return err
	}
	return nil
}

// isLocked should only be called by functions that hold a read lock on km!
func (km *keyManager) isLocked() bool {
	return len(km.password) == 0
}

func getFieldNameForKey(unknownKey Key) (string, error) {
	switch unknownKey.(type) {
	case csakey.KeyV2:
		return "CSA", nil
	case ethkey.KeyV2:
		return "Eth", nil
	case ocrkey.KeyV2:
		return "OCR", nil
	case p2pkey.KeyV2:
		return "P2P", nil
	case vrfkey.KeyV2:
		return "VRF", nil
	}
	return "", errors.New(fmt.Sprintf("Unknown key type: %T", unknownKey))
}

func (km *keyManager) withDB(db *gorm.DB) *keyManager {
	return &keyManager{
		ksORM:        NewORM(db),
		scryptParams: km.scryptParams,
		keyRing:      km.keyRing,
		lock:         km.lock,
		password:     km.password,
	}
}

type Key interface {
	ID() string
}
