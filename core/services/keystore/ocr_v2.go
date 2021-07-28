package keystore

import (
	"encoding/hex"
	"fmt"

	p2ppeer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/ocrkey"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/p2pkey"
	"github.com/smartcontractkit/chainlink/core/store/models"
)

type OCR interface {
	// P2P Keys
	DecryptedP2PKey(peerID p2ppeer.ID) (p2pkey.KeyV2, bool)
	DecryptedP2PKeys() (keys []p2pkey.KeyV2)
	GenerateP2PKey() (p2pkey.KeyV2, error)
	GetP2PKeys() (keys []p2pkey.KeyV2, err error)
	GetP2PKey(id string) (*p2pkey.KeyV2, error)
	FindEncryptedP2PKeyByID(id int32) (*p2pkey.KeyV2, error)
	ArchiveEncryptedP2PKey(key *p2pkey.KeyV2) error
	DeleteP2PKey(key *p2pkey.KeyV2) error
	ImportP2PKey(keyJSON []byte, oldPassword string) (*p2pkey.KeyV2, error)
	ExportP2PKey(ID int32, newPassword string) ([]byte, error)
	// OCR Keys
	DecryptedOCRKey(hash models.Sha256Hash) (ocrkey.KeyV2, bool)
	GenerateOCRKey() (ocrkey.KeyV2, error)
	UpsertEncryptedOCRKeyBundle(encryptedKey *ocrkey.KeyV2) error
	FindEncryptedOCRKeyBundles() (keys []ocrkey.KeyV2, err error)
	GetOCRKey(id string) (ocrkey.KeyV2, error)
	ArchiveEncryptedOCRKeyBundle(key *ocrkey.KeyV2) error
	DeleteEncryptedOCRKeyBundle(key *ocrkey.KeyV2) error
	ImportOCRKeyBundle(keyJSON []byte, oldPassword string) (*ocrkey.KeyV2, error)
	ExportOCRKeyBundle(id models.Sha256Hash, newPassword string) ([]byte, error)
}

type ocr struct {
	*keyManager
}

var _ OCR = ocr{}

func newOCRKeyStore(km *keyManager) ocr {
	return ocr{
		km,
	}
}

func (ks ocr) DecryptedP2PKey(peerID p2ppeer.ID) (p2pkey.KeyV2, bool) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	k, exists := ks.keyRing.P2P[peerID.String()]
	return k, exists
}

func (ks ocr) DecryptedP2PKeys() (keys []p2pkey.KeyV2) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	for _, key := range ks.keyRing.P2P {
		keys = append(keys, key)
	}
	return keys
}

// TODO - change this signature to accept key ID type
func (ks ocr) DecryptedOCRKey(hash models.Sha256Hash) (ocrkey.KeyV2, bool) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	keyID := hex.EncodeToString(hash[:])
	k, exists := ks.keyRing.OCR[keyID]
	if !exists {
		return ocrkey.KeyV2{}, false
	}
	return k, true
}

func (ks ocr) GenerateP2PKey() (p2pkey.KeyV2, error) {
	ks.lock.Lock()
	defer ks.lock.Unlock()
	if ks.isLocked() {
		return p2pkey.KeyV2{}, LockedErr
	}
	key, err := p2pkey.NewV2()
	if err != nil {
		return p2pkey.KeyV2{}, errors.Wrapf(err, "while generating new p2p key")
	}
	err = ks.safeAddKey(key)
	if err != nil {
		return p2pkey.KeyV2{}, errors.Wrapf(err, "while adding new p2p key")
	}
	return key, nil
}

func (ks ocr) GetP2PKeys() (keys []p2pkey.KeyV2, err error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return keys, LockedErr
	}
	for _, key := range ks.keyRing.P2P {
		keys = append(keys, key)
	}
	return keys, nil
}

func (ks ocr) GetP2PKey(id string) (*p2pkey.KeyV2, error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return nil, LockedErr
	}
	key, found := ks.keyRing.P2P[id]
	if !found {
		return nil, errors.New(fmt.Sprintf("P2P key not found with ID %s", id))
	}
	return &key, nil
}

func (ks ocr) FindEncryptedP2PKeyByID(id int32) (*p2pkey.KeyV2, error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return nil, LockedErr
	}

	return nil, nil
}

func (ks ocr) ArchiveEncryptedP2PKey(key *p2pkey.KeyV2) error {
	return errors.New("hard delete only")
}

func (ks ocr) DeleteP2PKey(key *p2pkey.KeyV2) error {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return LockedErr
	}
	return ks.safeRemoveKey(key)
}

func (ks ocr) GenerateOCRKey() (ocrkey.KeyV2, error) {
	ks.lock.Lock()
	defer ks.lock.Unlock()
	if ks.isLocked() {
		return ocrkey.KeyV2{}, LockedErr
	}
	key, err := ocrkey.NewV2()
	if err != nil {
		return ocrkey.KeyV2{}, errors.Wrapf(err, "while generating new ocr key")
	}
	err = ks.safeAddKey(key)
	if err != nil {
		return ocrkey.KeyV2{}, errors.Wrapf(err, "while adding new ocr key")
	}
	return key, nil
}

func (ks ocr) UpsertEncryptedOCRKeyBundle(encryptedKey *ocrkey.KeyV2) error {
	return nil
}

func (ks ocr) FindEncryptedOCRKeyBundles() (keys []ocrkey.KeyV2, err error) {
	return keys, err
}

func (ks ocr) GetOCRKey(id string) (ocrkey.KeyV2, error) {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	if ks.isLocked() {
		return ocrkey.KeyV2{}, LockedErr
	}
	key, found := ks.keyRing.OCR[id]
	if !found {
		return ocrkey.KeyV2{}, errors.New(fmt.Sprintf("OCR key not found with ID %s", id))
	}
	return key, nil
}

func (ks ocr) ArchiveEncryptedOCRKeyBundle(key *ocrkey.KeyV2) error {
	return nil
}

func (ks ocr) DeleteEncryptedOCRKeyBundle(key *ocrkey.KeyV2) error {
	return nil
}

func (ks ocr) ImportP2PKey(keyJSON []byte, oldPassword string) (*p2pkey.KeyV2, error) {
	return nil, nil
}

func (ks ocr) ExportP2PKey(ID int32, newPassword string) ([]byte, error) {
	return []byte{}, nil
}

func (ks ocr) ImportOCRKeyBundle(keyJSON []byte, oldPassword string) (*ocrkey.KeyV2, error) {
	return nil, nil
}

func (ks ocr) ExportOCRKeyBundle(id models.Sha256Hash, newPassword string) ([]byte, error) {
	return []byte{}, nil
}
