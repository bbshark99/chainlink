package keystore

import (
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
	GenerateProof(k secp256k1.PublicKey, seed *big.Int) (vrfkey.Proof, error)
	Forget(k secp256k1.PublicKey) error
	CreateKey() (secp256k1.PublicKey, error)
	CreateAndUnlockWeakInMemoryEncryptedKeyXXXTestingOnly(phrase string) (*vrfkey.EncryptedVRFKey, error)
	Store(key *vrfkey.PrivateKey, phrase string, scryptParams utils.ScryptParams) error
	StoreInMemoryXXXTestingOnly(key *vrfkey.PrivateKey)
	Archive(key secp256k1.PublicKey) (err error)
	Delete(key secp256k1.PublicKey) (err error)
	Import(keyjson []byte, auth string) (vrfkey.EncryptedVRFKey, error)
	Export(pk secp256k1.PublicKey, newPassword string) ([]byte, error)
	Get(k ...secp256k1.PublicKey) ([]*vrfkey.EncryptedVRFKey, error)
	GetSpecificKey(k secp256k1.PublicKey) (*vrfkey.EncryptedVRFKey, error)
	ListKeys() (publicKeys []*secp256k1.PublicKey, err error)
	ListKeysIncludingArchived() (publicKeys []*secp256k1.PublicKey, err error)
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

func (ks vrf) GenerateProof(k secp256k1.PublicKey, seed *big.Int) (vrfkey.Proof, error) {
	return vrfkey.Proof{}, nil
}

func (ks vrf) Forget(k secp256k1.PublicKey) error {
	return nil
}

func (ks vrf) CreateKey() (secp256k1.PublicKey, error) {
	return secp256k1.PublicKey{}, nil
}

func (ks vrf) CreateAndUnlockWeakInMemoryEncryptedKeyXXXTestingOnly(phrase string) (*vrfkey.EncryptedVRFKey, error) {
	return nil, nil
}

func (ks vrf) Store(key *vrfkey.PrivateKey, phrase string, scryptParams utils.ScryptParams) error {
	return nil
}

func (ks vrf) StoreInMemoryXXXTestingOnly(key *vrfkey.PrivateKey) {

}

func (ks vrf) Archive(key secp256k1.PublicKey) (err error) {
	return nil
}

func (ks vrf) Delete(key secp256k1.PublicKey) (err error) {
	return nil
}

func (ks vrf) Import(keyjson []byte, auth string) (vrfkey.EncryptedVRFKey, error) {
	return vrfkey.EncryptedVRFKey{}, nil
}

func (ks vrf) Export(pk secp256k1.PublicKey, newPassword string) ([]byte, error) {
	return []byte{}, nil
}

func (ks vrf) Get(k ...secp256k1.PublicKey) ([]*vrfkey.EncryptedVRFKey, error) {
	return []*vrfkey.EncryptedVRFKey{}, nil
}

func (ks vrf) GetSpecificKey(k secp256k1.PublicKey) (*vrfkey.EncryptedVRFKey, error) {
	return nil, nil
}

func (ks vrf) ListKeys() (publicKeys []*secp256k1.PublicKey, err error) {
	return nil, nil
}

func (ks vrf) ListKeysIncludingArchived() (publicKeys []*secp256k1.PublicKey, err error) {
	return nil, nil
}
