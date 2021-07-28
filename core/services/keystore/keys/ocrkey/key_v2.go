package ocrkey

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink/core/utils"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting/types"
	"golang.org/x/crypto/curve25519"
)

// type ID string

type Raw []byte

var (
	ErrScalarTooBig = errors.Errorf("can't handle scalars greater than %d", curve25519.PointSize)
	curve           = secp256k1.S256()
)

// TODO - RYAN - Rehydrate() ?
func (rawKey Raw) Key() (key KeyV2) {
	ecdsaD := big.NewInt(0).SetBytes(rawKey[:32])
	var ed25519PrivKey []byte = rawKey[32:96]
	var offChainEncryption [32]byte
	copy(offChainEncryption[:], rawKey[96:])
	// var D []byte = rawKey[:32]

	// ecdsaDSize := len(rawKey.EcdsaD.Bytes())
	// if ecdsaDSize > curve25519.PointSize {
	// 	panic(errors.Wrapf(ErrScalarTooBig, "got %d byte ecdsa scalar", ecdsaDSize))
	// }

	publicKey := ecdsa.PublicKey{Curve: curve}
	publicKey.X, publicKey.Y = curve.ScalarBaseMult(ecdsaD.Bytes())
	privateKey := ecdsa.PrivateKey{
		PublicKey: publicKey,
		D:         ecdsaD,
	}
	OnChainSigning := onChainPrivateKey(privateKey)
	OffChainSigning := offChainPrivateKey(ed25519PrivKey)
	key.OnChainSigning = &OnChainSigning
	key.OffChainSigning = &OffChainSigning
	key.OffChainEncryption = &offChainEncryption
	// key.ID = generateID(&key)
	return key
}

// TODO - RYAN - check that all KeyV2 structs contain pointers to privKey material
type KeyV2 struct {
	OnChainSigning     *onChainPrivateKey
	OffChainSigning    *offChainPrivateKey
	OffChainEncryption *[curve25519.ScalarSize]byte
}

func NewV2() (KeyV2, error) {
	ecdsaKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return KeyV2{}, err
	}
	onChainPriv := (*onChainPrivateKey)(ecdsaKey)

	_, offChainPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyV2{}, err
	}
	var encryptionPriv [curve25519.ScalarSize]byte
	_, err = rand.Reader.Read(encryptionPriv[:])
	if err != nil {
		return KeyV2{}, err
	}
	k := KeyV2{
		OnChainSigning:     onChainPriv,
		OffChainSigning:    (*offChainPrivateKey)(&offChainPriv),
		OffChainEncryption: &encryptionPriv,
	}
	return k, nil
}

func (key KeyV2) ID() string {
	sha := sha256.Sum256(key.Raw())
	return hex.EncodeToString(sha[:])
}

func (key KeyV2) Raw() Raw {
	return utils.ConcatBytes(
		key.OnChainSigning.D.Bytes(),
		[]byte(*key.OffChainSigning),
		key.OffChainEncryption[:],
	)
}

// func (key KeyV2) ToKeyV1() KeyBundle {
// 	id := models.Sha256Hash{}
// 	idBytes, err := hex.DecodeString(key.ID())
// 	copy(id[:], idBytes)
// 	if err != nil {
// 		panic(errors.Wrap(err, "could not decode OCR key id bytes"))
// 	}
// 	return KeyBundle{
// 		ID:                 id,
// 		onChainSigning:     key.OnChainSigning,
// 		offChainSigning:    key.OffChainSigning,
// 		offChainEncryption: key.OffChainEncryption,
// 	}
// }

// SignOnChain returns an ethereum-style ECDSA secp256k1 signature on msg.
func (pk *KeyV2) SignOnChain(msg []byte) (signature []byte, err error) {
	return pk.OnChainSigning.Sign(msg)
}

// SignOffChain returns an EdDSA-Ed25519 signature on msg.
func (pk *KeyV2) SignOffChain(msg []byte) (signature []byte, err error) {
	return pk.OffChainSigning.Sign(msg)
}

// ConfigDiffieHellman returns the shared point obtained by multiplying someone's
// public key by a secret scalar ( in this case, the OffChainEncryption key.)
func (pk *KeyV2) ConfigDiffieHellman(base *[curve25519.PointSize]byte) (
	sharedPoint *[curve25519.PointSize]byte, err error,
) {
	p, err := curve25519.X25519(pk.OffChainEncryption[:], base[:])
	if err != nil {
		return nil, err
	}
	sharedPoint = new([ed25519.PublicKeySize]byte)
	copy(sharedPoint[:], p)
	return sharedPoint, nil
}

// TODO - RYAN - these function names suck
// PublicKeyAddressOnChain returns public component of the keypair used in
// SignOnChain
func (pk *KeyV2) PublicKeyAddressOnChain() ocrtypes.OnChainSigningAddress {
	return ocrtypes.OnChainSigningAddress(pk.OnChainSigning.Address())
}

// PublicKeyOffChain returns the pbulic component of the keypair used in SignOffChain
func (pk *KeyV2) PublicKeyOffChain() ocrtypes.OffchainPublicKey {
	return ocrtypes.OffchainPublicKey(pk.OffChainSigning.PublicKey())
}

// PublicKeyConfig returns the public component of the keypair used in ConfigKeyShare
func (pk *KeyV2) PublicKeyConfig() [curve25519.PointSize]byte {
	rv, err := curve25519.X25519(pk.OffChainEncryption[:], curve25519.Basepoint)
	if err != nil {
		log.Println("failure while computing public key: " + err.Error())
	}
	var rvFixed [curve25519.PointSize]byte
	copy(rvFixed[:], rv)
	return rvFixed
}
