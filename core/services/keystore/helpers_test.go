package keystore

import (
	"testing"

	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/csakey"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/ethkey"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/ocrkey"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/p2pkey"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/vrfkey"
	"github.com/smartcontractkit/chainlink/core/utils"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func mustNewEthKey(t *testing.T) ethkey.KeyV2 {
	key, err := ethkey.NewV2()
	require.NoError(t, err)
	return key
}

func mustNewCSAKey(t *testing.T) csakey.KeyV2 {
	key, err := csakey.NewV2()
	require.NoError(t, err)
	return key
}

func mustNewOCRKey(t *testing.T) ocrkey.KeyV2 {
	key, err := ocrkey.NewV2()
	require.NoError(t, err)
	return key
}

func mustNewP2PKey(t *testing.T) p2pkey.KeyV2 {
	key, err := p2pkey.NewV2()
	require.NoError(t, err)
	return key
}

func mustNewVRFKey(t *testing.T) vrfkey.KeyV2 {
	key, err := vrfkey.NewV2()
	require.NoError(t, err)
	return key
}

type ExportedEncryptedKeyRing = encryptedKeyRing

func NewMasterV2(db *gorm.DB) *masterV2 {
	return newV2(db, utils.FastScryptParams)
}

func (m *masterV2) ExportedSave() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.save()
}

func (m *masterV2) ResetXXXTestOnly() {
	keyRing := newKeyRing()
	m.keyRing = &keyRing
	m.password = ""
}
