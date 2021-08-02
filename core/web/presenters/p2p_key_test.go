package presenters

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/manyminds/api2go/jsonapi"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/p2pkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestP2PKeyResource(t *testing.T) {
	// _, pubKey, err := cryptop2p.GenerateEd25519Key(rand.Reader)
	// require.NoError(t, err)
	// pubKeyBytes, err := pubKey.Raw()
	// require.NoError(t, err)

	// peerIDStr := "12D3KooWFX81q1r31xnoQwJ4WdptssdEYZXcFAryonqK1Eo2XSvG"
	// p2pPeerID, err := peer.Decode(peerIDStr)
	// require.NoError(t, err)
	// peerID := p2pkey.PeerID(p2pPeerID)

	key, err := p2pkey.NewV2()
	require.NoError(t, err)
	peerID := key.PeerID()
	peerIDStr := peerID.String()
	pubKey := key.GetPublic()
	pubKeyBytes, err := pubKey.Raw()
	require.NoError(t, err)

	r := NewP2PKeyResource(key)
	b, err := jsonapi.Marshal(r)
	require.NoError(t, err)

	expected := fmt.Sprintf(`
	{
		"data":{
			"type":"encryptedP2PKeys",
			"id":"%s",
			"attributes":{
				"peerId":"%s",
				"publicKey": "%s"
			}
		}
	}`, key.ID(), peerIDStr, hex.EncodeToString(pubKeyBytes))

	assert.JSONEq(t, expected, string(b))

	r = NewP2PKeyResource(key)
	b, err = jsonapi.Marshal(r)
	require.NoError(t, err)

	expected = fmt.Sprintf(`
	{
		"data": {
			"type":"encryptedP2PKeys",
			"id":"%s",
			"attributes":{
				"peerId":"%s",
				"publicKey": "%s"
			}
		}
	}`, key.ID(), peerIDStr, hex.EncodeToString(pubKeyBytes))

	assert.JSONEq(t, expected, string(b))
}
