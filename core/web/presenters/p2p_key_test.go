package presenters

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"

	cryptop2p "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/p2pkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestP2PKeyResource(t *testing.T) {
	_, pubKey, err := cryptop2p.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)
	pubKeyBytes, err := pubKey.Raw()
	require.NoError(t, err)

	// peerIDStr := "12D3KooWApUJaQB2saFjyEUfq6BmysnsSnhLnY5CF9tURYVKgoXK"
	// p2pPeerID, err := peer.Decode(peerIDStr)
	// require.NoError(t, err)
	// peerID := p2pkey.PeerID(p2pPeerID)

	key, err := p2pkey.NewV2()
	require.NoError(t, err)
	peerID := key.PeerID()
	peerIDStr := peerID.String()

	r := NewP2PKeyResource(key)
	b, err := jsonapi.Marshal(r)
	require.NoError(t, err)

	expected := fmt.Sprintf(`
	{
		"data":{
			"type":"encryptedP2PKeys",
			"id":"1",
			"attributes":{
				"peerId":"%s",
				"publicKey": "%s",
				"createdAt":"2000-01-01T00:00:00Z",
				"updatedAt":"2000-01-01T00:00:00Z",
				"deletedAt":null
			}
		}
	}`, peerIDStr, hex.EncodeToString(pubKeyBytes))

	assert.JSONEq(t, expected, string(b))

	r = NewP2PKeyResource(key)
	b, err = jsonapi.Marshal(r)
	require.NoError(t, err)

	expected = fmt.Sprintf(`
	{
		"data": {
			"type":"encryptedP2PKeys",
			"id":"1",
			"attributes":{
				"peerId":"p2p_%s",
				"publicKey": "%s",
			}
		}
	}`, peerIDStr, hex.EncodeToString(pubKeyBytes))

	assert.JSONEq(t, expected, string(b))
}
