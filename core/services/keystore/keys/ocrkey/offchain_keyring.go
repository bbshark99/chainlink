package ocrkey

import (
	"crypto/ed25519"

	"golang.org/x/crypto/curve25519"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2/types"
)

var _ ocrtypes.OffchainKeyring = &OffchainKeyring{}

type OffchainKeyring struct {
	signingKey    ed25519.PrivateKey
	encryptionKey ed25519.PrivateKey
}

func (ok *OffchainKeyring) OffchainSign(msg []byte) (signature []byte, err error) {
	return ed25519.Sign(ed25519.PrivateKey(ok.signingKey), msg), nil
}

func (ok *OffchainKeyring) ConfigDiffieHellman(point [curve25519.PointSize]byte) (sharedPoint [curve25519.PointSize]byte, err error) {
	p, err := curve25519.X25519(ok.signingKey[:], point[:])
	if err != nil {
		return
	}
	sharedPoint = [ed25519.PublicKeySize]byte{}
	copy(sharedPoint[:], p)
	return
}

func (ok *OffchainKeyring) OffchainPublicKey() ocrtypes.OffchainPublicKey {
	return ocrtypes.OffchainPublicKey(ed25519.PrivateKey(ok.signingKey).Public().(ed25519.PublicKey))
}

func (ok *OffchainKeyring) ConfigEncryptionPublicKey() ocrtypes.ConfigEncryptionPublicKey {
	return ocrtypes.ConfigEncryptionPublicKey(ed25519.PrivateKey(ok.encryptionKey).Public().([curve25519.PointSize]byte))
}
