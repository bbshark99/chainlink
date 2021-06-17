package ethkey

import (
	"crypto/ecdsa"
	"crypto/rand"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
)

var curve = crypto.S256()

type Raw []byte

func (raw Raw) Key() KeyV2 {
	var privateKey ecdsa.PrivateKey
	d := big.NewInt(0).SetBytes(raw)
	privateKey.PublicKey.Curve = curve
	privateKey.D = d
	privateKey.PublicKey.X, privateKey.PublicKey.Y = curve.ScalarBaseMult(d.Bytes())
	address := EIP55AddressFromAddress(crypto.PubkeyToAddress(privateKey.PublicKey))
	return KeyV2{
		Address:    address,
		privateKey: privateKey,
	}
}

type KeyV2 struct {
	Address    EIP55Address
	privateKey ecdsa.PrivateKey
}

func NewV2() (key KeyV2, err error) {
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return key, err
	}
	return FromPrivateKey(privateKeyECDSA), nil
}

func FromPrivateKey(privKey *ecdsa.PrivateKey) (key KeyV2) {
	address := EIP55AddressFromAddress(crypto.PubkeyToAddress(privKey.PublicKey))
	key = KeyV2{
		Address:    address,
		privateKey: *privKey,
	}
	return key
}

func (key KeyV2) ID() string {
	return key.Address.Hex()
}

func (key KeyV2) Raw() Raw {
	return key.privateKey.D.Bytes()
}

func (key KeyV2) ToEcdsaPrivKey() *ecdsa.PrivateKey {
	return &key.privateKey
}

func (key KeyV2) ToKeyV1() Key {
	return Key{
		Address: key.Address,
	}
}

func (key KeyV2) ToGethKey() keystore.Key {
	return keystore.Key{
		Address:    key.Address.Address(),
		PrivateKey: &key.privateKey,
	}
}
