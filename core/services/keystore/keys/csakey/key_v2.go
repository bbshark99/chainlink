package csakey

import (
	"crypto/ed25519"
	cryptorand "crypto/rand"
	"encoding/hex"
)

type Raw []byte

func (rawKey Raw) Key() KeyV2 {
	privKey := ed25519.PrivateKey(rawKey)
	return KeyV2{
		privateKey: privKey,
		PublicKey:  ed25519PubKeyFromPrivKey(privKey),
	}
}

type KeyV2 struct {
	privateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	Version    int
}

func NewV2() (KeyV2, error) {
	pubKey, privKey, err := ed25519.GenerateKey(cryptorand.Reader)
	if err != nil {
		return KeyV2{}, err
	}
	return KeyV2{
		privateKey: privKey,
		PublicKey:  pubKey,
		Version:    2,
	}, nil
}

func (key KeyV2) ID() string {
	return key.PublicKeyString()
}

func (key KeyV2) PublicKeyString() string {
	return hex.EncodeToString(key.PublicKey)
}

func (key KeyV2) Raw() Raw {
	return Raw(key.privateKey)
}

func ed25519PubKeyFromPrivKey(privKey ed25519.PrivateKey) ed25519.PublicKey {
	publicKey := make([]byte, ed25519.PublicKeySize)
	copy(publicKey, privKey[32:])
	return ed25519.PublicKey(publicKey)
}
