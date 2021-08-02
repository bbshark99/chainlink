package keystore_test

import (
	"testing"

	"github.com/smartcontractkit/chainlink/core/internal/cltest"
	"github.com/smartcontractkit/chainlink/core/services/keystore"
	"github.com/stretchr/testify/require"
)

func TestMasterKeystoreV2_Unlock_Save(t *testing.T) {
	t.Parallel()
	store, cleanup := cltest.NewStore(t) // TODO - remove store
	defer cleanup()
	db := store.DB
	keyStore := keystore.NewMasterV2(db)
	reset := func() {
		keyStore.ResetXXXTestOnly()
		err := db.Exec("DELETE FROM encrypted_key_rings").Error
		require.NoError(t, err)
		cltest.AssertCount(t, db, keystore.ExportedEncryptedKeyRing{}, 0)
	}

	t.Run("test database initializes with fixtures and default password", func(t *testing.T) {
		defer reset()
		require.Error(t, keyStore.Unlock("wrong password"))
		require.NoError(t, keyStore.Unlock(cltest.Password))
		ocrKeys, err := keyStore.OCR().GetOCRKeys()
		require.NoError(t, err)
		require.Equal(t, 1, len(ocrKeys))
		require.Equal(t, "2dec5de7aff8164412c0fbaa2f06654e10e709ee78f031cba9244d453399358e", ocrKeys[0].ID())
		p2pKeys, err := keyStore.OCR().GetP2PKeys()
		require.NoError(t, err)
		require.Equal(t, 1, len(p2pKeys))
		require.Equal(t, "12D3KooWFX81q1r31xnoQwJ4WdptssdEYZXcFAryonqK1Eo2XSvG", p2pKeys[0].ID())
	})

	t.Run("unlock creates a new encryptedKeyRing with ID 1", func(t *testing.T) {
		defer reset()
		cltest.AssertCount(t, db, keystore.ExportedEncryptedKeyRing{}, 0)
		require.NoError(t, keyStore.Unlock(cltest.Password))
		var count int64
		db.Table("encrypted_key_rings").Where("id = ?", 1).Count(&count)
		require.Equal(t, int64(1), count)
	})

	t.Run("can be unlocked more than once, as long as the passwords match", func(t *testing.T) {
		defer reset()
		require.NoError(t, keyStore.Unlock(cltest.Password))
		require.NoError(t, keyStore.Unlock(cltest.Password))
		require.NoError(t, keyStore.Unlock(cltest.Password))
		require.Error(t, keyStore.Unlock("wrong password"))
	})

	t.Run("saves an empty keyRing", func(t *testing.T) {
		defer reset()
		require.NoError(t, keyStore.Unlock(cltest.Password))
		cltest.AssertCount(t, db, keystore.ExportedEncryptedKeyRing{}, 1)
		require.NoError(t, keyStore.ExportedSave())
		cltest.AssertCount(t, db, keystore.ExportedEncryptedKeyRing{}, 1)
	})

	t.Run("won't load a saved keyRing if the password is incorrect", func(t *testing.T) {
		defer reset()
		require.NoError(t, keyStore.Unlock(cltest.Password))
		require.NoError(t, keyStore.ExportedSave())
		keyStore.ResetXXXTestOnly()
		require.Error(t, keyStore.Unlock("password2"))
	})

	t.Run("loads a saved keyRing if the password is correct", func(t *testing.T) {
		defer reset()
		require.NoError(t, keyStore.Unlock(cltest.Password))
		require.NoError(t, keyStore.ExportedSave())
		keyStore.ResetXXXTestOnly()
		require.NoError(t, keyStore.Unlock(cltest.Password))
	})

	// err := db.Exec("DELETE FROM encrypted_key_rings").Error
	// require.NoError(t, err)
	// require.NoError(t, keyStore.Unlock(cltest.Password))
	// // TODO - RYAN - lawl these should all be the same function name
	// ocrkey, err := keyStore.OCR().GenerateOCRKey()
	// require.NoError(t, err)
	// fmt.Println("ocrkey.ID", ocrkey.ID())
	// p2pkey, err := keyStore.OCR().GenerateP2PKey()
	// require.NoError(t, err)
	// err = keyStore.ExportedSave()
	// require.NoError(t, err)
	// fmt.Println("p2pkey.ID", p2pkey.ID())
}
