package keystore_test

import (
	"testing"
	"time"

	"github.com/smartcontractkit/chainlink/core/internal/cltest"
	"github.com/smartcontractkit/chainlink/core/services/keystore"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/ethkey"
	"github.com/stretchr/testify/require"
)

func TestEthKeyStore_V2(t *testing.T) {
	t.Parallel()

	store, cleanup := cltest.NewStore(t)
	defer cleanup()
	db := store.DB
	keyStore := keystore.NewMasterV2(db)
	keyStore.Unlock(cltest.Password)
	ethKeyStore := keyStore.Eth()
	reset := func() {
		keyStore.ResetXXXTestOnly()
		require.NoError(t, store.DB.Exec("DELETE FROM encrypted_key_rings").Error)
		require.NoError(t, store.DB.Exec("DELETE FROM eth_key_states").Error)
		keyStore.Unlock(cltest.Password)
	}

	t.Run("CreateNewKey / AllKeys / AllStates / KeyByAddress", func(t *testing.T) {
		defer reset()
		key, err := ethKeyStore.CreateNewKey()
		require.NoError(t, err)
		retrievedKeys, err := ethKeyStore.AllKeys()
		require.NoError(t, err)
		require.Equal(t, 1, len(retrievedKeys))
		require.Equal(t, key.Address, retrievedKeys[0].Address)
		foundKey, err := ethKeyStore.KeyByAddress(key.Address.Address())
		require.NoError(t, err)
		require.Equal(t, key, foundKey)
		// adds ethkey.State
		cltest.AssertCount(t, store.DB, ethkey.State{}, 1)
		var state ethkey.State
		require.NoError(t, db.First(&state).Error)
		require.Equal(t, state.Address, retrievedKeys[0].Address)
		// adds key to db
		keyStore.ResetXXXTestOnly()
		keyStore.Unlock(cltest.Password)
		retrievedKeys, err = ethKeyStore.AllKeys()
		require.NoError(t, err)
		require.Equal(t, 1, len(retrievedKeys))
		require.Equal(t, key.Address, retrievedKeys[0].Address)
		// adds 2nd key
		_, err = ethKeyStore.CreateNewKey()
		require.NoError(t, err)
		retrievedKeys, err = ethKeyStore.AllKeys()
		require.NoError(t, err)
		require.Equal(t, 2, len(retrievedKeys))
	})

	t.Run("RemoveKey", func(t *testing.T) {
		defer reset()
		key, err := ethKeyStore.CreateNewKey()
		require.NoError(t, err)
		_, err = ethKeyStore.RemoveKey(key.Address.Address(), false) // no soft delete
		require.Error(t, err)
		_, err = ethKeyStore.RemoveKey(key.Address.Address(), true)
		require.NoError(t, err)
		retrievedKeys, err := ethKeyStore.AllKeys()
		require.NoError(t, err)
		require.Equal(t, 0, len(retrievedKeys))
		cltest.AssertCount(t, store.DB, ethkey.State{}, 0)
	})

	t.Run("SendingKeys / HasSendingKeyWithAddress / HasDBSendingKeys", func(t *testing.T) {
		defer reset()
		has, err := ethKeyStore.HasDBSendingKeys()
		require.NoError(t, err)
		require.False(t, has)
		key, err := ethKeyStore.CreateNewKey()
		require.NoError(t, err)
		has, err = ethKeyStore.HasDBSendingKeys()
		require.NoError(t, err)
		require.True(t, has)
		sendingKeys, err := ethKeyStore.SendingKeys()
		require.NoError(t, err)
		require.Equal(t, 1, len(sendingKeys))
		require.Equal(t, key.Address, sendingKeys[0].Address)
		fundingKeys, err := ethKeyStore.FundingKeys()
		require.NoError(t, err)
		require.Equal(t, 0, len(fundingKeys))
		cltest.AssertCount(t, store.DB, ethkey.State{}, 1)
		has, err = ethKeyStore.HasSendingKeyWithAddress(key.Address.Address())
		require.NoError(t, err)
		require.True(t, has)
		_, err = ethKeyStore.RemoveKey(key.Address.Address(), true)
		require.NoError(t, err)
		cltest.AssertCount(t, store.DB, ethkey.State{}, 0)
		has, err = ethKeyStore.HasSendingKeyWithAddress(key.Address.Address())
		require.NoError(t, err)
		require.False(t, has)
	})

	t.Run("EnsureFundingKey / FundingKeys / IsFundingKey", func(t *testing.T) {
		defer reset()
		key, didExist, err := ethKeyStore.EnsureFundingKey()
		require.NoError(t, err)
		require.False(t, didExist)
		fundingKeys, err := ethKeyStore.FundingKeys()
		require.NoError(t, err)
		require.Equal(t, 1, len(fundingKeys))
		require.Equal(t, key.Address, fundingKeys[0].Address)
		sendingKeys, err := ethKeyStore.SendingKeys()
		require.NoError(t, err)
		require.Equal(t, 0, len(sendingKeys))
		cltest.AssertCount(t, store.DB, ethkey.State{}, 1)
		is, err := ethKeyStore.IsFundingKey(key)
		require.NoError(t, err)
		require.True(t, is)
		key2, err := ethKeyStore.CreateNewKey()
		require.NoError(t, err)
		is, err = ethKeyStore.IsFundingKey(key2)
		require.NoError(t, err)
	})

	t.Run("GetRoundRobinAddress", func(t *testing.T) {
		defer reset()
		// should error when no addresses
		_, err := ethKeyStore.GetRoundRobinAddress()
		require.Error(t, err)
		_, _, err = ethKeyStore.EnsureFundingKey()
		require.NoError(t, err)
		// should error when no sending addresses
		_, err = ethKeyStore.GetRoundRobinAddress()
		require.Error(t, err)
		// should succeed when address present
		key, err := ethKeyStore.CreateNewKey()
		require.NoError(t, err)
		address, err := ethKeyStore.GetRoundRobinAddress()
		require.NoError(t, err)
		require.Equal(t, key.Address.Address(), address)
		err = db.Model(ethkey.State{}).
			Where("address = ?", key.Address).
			Update("last_used", time.Now().Add(-time.Hour)). // 1h ago
			Error
		require.NoError(t, err)
		// add 2nd key
		key2, err := ethKeyStore.CreateNewKey()
		require.NoError(t, err)
		err = db.Model(ethkey.State{}).
			Where("address = ?", key2.Address).
			Update("last_used", time.Now().Add(-2*time.Hour)). // 2h ago
			Error
		require.NoError(t, err)
		address, err = ethKeyStore.GetRoundRobinAddress()
		require.NoError(t, err)
		require.Equal(t, key2.Address.Address(), address)
		err = db.Model(ethkey.State{}).
			Where("address = ?", key2.Address).
			Update("last_used", time.Now().Add(-10*time.Minute)). // 10 min ago
			Error
		require.NoError(t, err)
		address, err = ethKeyStore.GetRoundRobinAddress()
		require.NoError(t, err)
		require.Equal(t, key.Address.Address(), address)
		// with a whitelist
		address, err = ethKeyStore.GetRoundRobinAddress(key2.Address.Address(), cltest.NewAddress())
		require.NoError(t, err)
		require.Equal(t, key2.Address.Address(), address)
		//  should error when no keys match whitelist
		address, err = ethKeyStore.GetRoundRobinAddress(cltest.NewAddress(), cltest.NewAddress())
		require.Error(t, err)
	})
}
