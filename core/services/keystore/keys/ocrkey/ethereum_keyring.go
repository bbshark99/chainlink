package ocrkey

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2/types"
)

var _ ocrtypes.OnchainKeyring = &EthereumKeyring{}

type EthereumKeyring struct {
	privateKey ecdsa.PrivateKey
}

func (ok *EthereumKeyring) PublicKey() ocrtypes.OnchainPublicKey {
	publicKey := (*ecdsa.PrivateKey)(&ok.privateKey).Public().(ecdsa.PublicKey)
	bytes := crypto.FromECDSAPub(&publicKey)
	return ocrtypes.OnchainPublicKey(bytes)
}

func (ok *EthereumKeyring) Sign(reportCtx ocrtypes.ReportContext, report ocrtypes.Report) (signature []byte, err error) {
	// TODO: how do we encode the message?
	msg := []byte{}
	sig, err := crypto.Sign(onChainHash(msg), (*ecdsa.PrivateKey)(&ok.privateKey))
	return sig, err
}

func (ok *EthereumKeyring) Verify(_ ocrtypes.OnchainPublicKey, _ ocrtypes.ReportContext, _ ocrtypes.Report, signature []byte) bool {
	// TODO: implement
	return true
}

func (ok *EthereumKeyring) MaxSignatureLength() int {
	// TODO: implement
	return 0
}
