package ethkey

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink/core/utils"
	"go.uber.org/multierr"
)

const keyTypeIdentifier = "Eth"

func FromEncryptedJSON(keyJSON []byte, password string) (KeyV2, error) {
	var key KeyV2
	var merr error
	var err2 error
	// TODO - remove once keystore V1 is removed
	key, err := decryptV1(keyJSON, password)
	if err != nil {
		merr = multierr.Append(merr, err)
		key, err2 = decryptV2(keyJSON, password)
		if err2 != nil {
			merr = multierr.Append(merr, err2)
			merr = multierr.Append(merr, errors.New("invalid key json"))
			return KeyV2{}, merr
		}
	}
	return key, nil
}

type EncryptedEthKeyExport struct {
	KeyType string              `json:"keyType"`
	Address EIP55Address        `json:"address"`
	Crypto  keystore.CryptoJSON `json:"crypto"`
}

func (key KeyV2) ToEncryptedJSON(password string, scryptParams utils.ScryptParams) (export []byte, err error) {
	cryptoJSON, err := keystore.EncryptDataV3(
		key.Raw(),
		[]byte(adulteratedPassword(password)),
		scryptParams.N,
		scryptParams.P,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not encrypt Eth key")
	}
	encryptedOCRKExport := EncryptedEthKeyExport{
		KeyType: keyTypeIdentifier,
		Address: key.Address,
		Crypto:  cryptoJSON,
	}
	return json.Marshal(encryptedOCRKExport)
}

// TODO - remove once keystore V1 is removed
func decryptV1(keyJSON []byte, password string) (KeyV2, error) {
	dKey, err := keystore.DecryptKey(keyJSON, password)
	if err != nil {
		return KeyV2{}, errors.Wrap(err, "failed to decrypt key as V1 type")
	}
	return KeyV2{
		Address:    EIP55AddressFromAddress(dKey.Address),
		privateKey: *dKey.PrivateKey,
	}, nil
}

func decryptV2(keyJSON []byte, password string) (KeyV2, error) {
	var export EncryptedEthKeyExport
	if err := json.Unmarshal(keyJSON, &export); err != nil {
		return KeyV2{}, err
	}
	privKey, err := keystore.DecryptDataV3(export.Crypto, adulteratedPassword(password))
	if err != nil {
		return KeyV2{}, errors.Wrap(err, "failed to decrypt key as V2 type")
	}
	return Raw(privKey).Key(), nil
}

func adulteratedPassword(password string) string {
	return "ethkey" + password
}
