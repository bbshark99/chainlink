package p2pkey

import (
	"crypto/rand"

	cryptop2p "github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
)

type Raw []byte

func (rawKey Raw) Key() KeyV2 {
	privKey, err := cryptop2p.UnmarshalPrivateKey(rawKey)
	if err != nil {
		panic(err) // TODO - RYAN - uhhhmmmm?
	}
	key, err := fromPrivkey(privKey)
	if err != nil {
		panic(err)
	}
	return key
}

type KeyV2 struct {
	cryptop2p.PrivKey // TODO - RYAN embed?
	peerID            PeerID
}

func NewV2() (KeyV2, error) {
	privKey, _, err := cryptop2p.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return KeyV2{}, nil
	}
	return fromPrivkey(privKey)
}

func (key KeyV2) ID() string {
	return peer.ID(key.peerID).String()
}

func (key KeyV2) Raw() Raw {
	marshalledPrivK, err := cryptop2p.MarshalPrivateKey(key.PrivKey)
	if err != nil {
		panic(err)
	}
	return marshalledPrivK
}

func (key KeyV2) ToKeyV1() Key {
	return Key{
		PrivKey: key.PrivKey,
	}
}

func (key KeyV2) ToKeyEncryptedV1() EncryptedP2PKey {
	pubKey, err := key.PrivKey.GetPublic().Bytes()
	if err != nil {
		panic(err)
	}
	return EncryptedP2PKey{
		PeerID: key.peerID,
		PubKey: PublicKeyBytes(pubKey),
	}
}

func (k KeyV2) GetPeerID() (PeerID, error) {
	peerID, err := peer.IDFromPrivateKey(k.PrivKey)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return PeerID(peerID), err
}

func (k KeyV2) MustGetPeerID() PeerID {
	peerID, err := k.GetPeerID()
	if err != nil {
		panic(err)
	}
	return peerID
}

func fromPrivkey(privKey cryptop2p.PrivKey) (KeyV2, error) {
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return KeyV2{}, err
	}
	return KeyV2{
		PrivKey: privKey,
		peerID:  PeerID(peerID),
	}, nil
}
