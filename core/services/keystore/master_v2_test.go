package keystore_test

import (
	"testing"

	"github.com/smartcontractkit/chainlink/core/internal/cltest"
	"github.com/smartcontractkit/chainlink/core/services/keystore"
	"github.com/stretchr/testify/require"
)

func TestMasterKeystoreV2_Unlock_Save(t *testing.T) {
	t.Parallel()
	store, cleanup := cltest.NewStore(t)
	defer cleanup()
	masterKeystore := keystore.NewMasterV2(store.DB)
	reset := func() {
		masterKeystore.ResetXXXTestOnly()
		err := store.DB.Exec("DELETE FROM encrypted_key_rings").Error
		require.NoError(t, err)
		cltest.AssertCount(t, store.DB, keystore.ExportedEncryptedKeyRing{}, 0)
	}

	t.Run("unlock creates a new encryptedKeyRing with ID 1", func(t *testing.T) {
		defer reset()
		cltest.AssertCount(t, store.DB, keystore.ExportedEncryptedKeyRing{}, 0)
		require.NoError(t, masterKeystore.Unlock(cltest.Password))
		var count int64
		store.DB.Table("encrypted_key_rings").Where("id = ?", 1).Count(&count)
		require.Equal(t, int64(1), count)
	})

	t.Run("cannot be unlocked twice", func(t *testing.T) {
		defer reset()
		require.NoError(t, masterKeystore.Unlock(cltest.Password))
		require.Error(t, masterKeystore.Unlock(cltest.Password))
	})

	t.Run("saves an empty keyRing", func(t *testing.T) {
		defer reset()
		require.NoError(t, masterKeystore.Unlock(cltest.Password))
		cltest.AssertCount(t, store.DB, keystore.ExportedEncryptedKeyRing{}, 1)
		require.NoError(t, masterKeystore.ExportedSave())
		cltest.AssertCount(t, store.DB, keystore.ExportedEncryptedKeyRing{}, 1)
	})

	t.Run("won't load a saved keyRing if the password is incorrect", func(t *testing.T) {
		defer reset()
		require.NoError(t, masterKeystore.Unlock(cltest.Password))
		require.NoError(t, masterKeystore.ExportedSave())
		masterKeystore.ResetXXXTestOnly()
		require.Error(t, masterKeystore.Unlock("password2"))
	})

	t.Run("loads a saved keyRing if the password is correct", func(t *testing.T) {
		defer reset()
		require.NoError(t, masterKeystore.Unlock(cltest.Password))
		require.NoError(t, masterKeystore.ExportedSave())
		masterKeystore.ResetXXXTestOnly()
		require.NoError(t, masterKeystore.Unlock(cltest.Password))
	})
}
