package vrfkey

import (
	"math/big"
)

func MustNewPrivateKey(rawKey *big.Int) *KeyV2 {
	k, err := NewV2()
	if err != nil {
		panic(err)
	}
	return &k
}
