package vrfkey

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/smartcontractkit/chainlink/core/services/signatures/secp256k1"
	"go.dedis.ch/kyber/v3"
)

type Raw []byte

func (rawKey Raw) Key() KeyV2 {
	rawKeyInt := new(big.Int).SetBytes(rawKey)
	k := secp256k1.IntToScalar(rawKeyInt)
	key, err := keyFromScalar(k)
	if err != nil {
		panic(err)
	}
	return key
}

type KeyV2 struct {
	k         kyber.Scalar
	PublicKey secp256k1.PublicKey
}

func NewV2() (KeyV2, error) {
	k := suite.Scalar().Pick(suite.RandomStream())
	return keyFromScalar(k)
}

func (key KeyV2) ID() string {
	return hex.EncodeToString(key.PublicKey[:])
}

func (key KeyV2) Raw() Raw {
	return secp256k1.ToInt(key.k).Bytes()
}

func keyFromScalar(k kyber.Scalar) (KeyV2, error) {
	rawPublicKey, err := secp256k1.ScalarToPublicPoint(k).MarshalBinary()
	if err != nil {
		return KeyV2{}, err
	}
	if len(rawPublicKey) != secp256k1.CompressedPublicKeyLength {
		return KeyV2{}, fmt.Errorf("public key %x has wrong length", rawPublicKey)
	}
	var publicKey secp256k1.PublicKey
	copy(publicKey[:], rawPublicKey)
	return KeyV2{
		k:         k,
		PublicKey: publicKey,
	}, nil
}
