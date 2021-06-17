package fluxmonitorv2_test

import (
	"testing"

	"github.com/smartcontractkit/chainlink/core/internal/cltest"
	"github.com/smartcontractkit/chainlink/core/services/fluxmonitorv2"
	"github.com/stretchr/testify/require"
)

func TestKeyStore_SendingKeys(t *testing.T) {
	t.Parallel()

	s, cleanup := cltest.NewStore(t)
	t.Cleanup(cleanup)
	ethKeyStore := cltest.NewKeyStore(t, s.DB).Eth()

	ks := fluxmonitorv2.NewKeyStore(ethKeyStore)

	key, err := ethKeyStore.CreateNewKey()
	require.NoError(t, err)

	keys, err := ks.SendingKeys()
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Equal(t, key, keys[0])
}

func TestKeyStore_GetRoundRobinAddress(t *testing.T) {
	t.Parallel()

	s, cleanup := cltest.NewStore(t)
	t.Cleanup(cleanup)
	ethKeyStore := cltest.NewKeyStore(t, s.DB).Eth()

	cltest.MustInsertRandomKey(t, s.DB, ethKeyStore, 0, true)
	_, k0Address := cltest.MustInsertRandomKey(t, s.DB, ethKeyStore, 0)

	ks := fluxmonitorv2.NewKeyStore(ethKeyStore)

	// Gets the only address in the keystore
	addr, err := ks.GetRoundRobinAddress()
	require.NoError(t, err)
	require.Equal(t, k0Address, addr)
}
