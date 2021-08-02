package vrfkey

// var sk = 0xdeadbeefdeadbee
// var k = MustNewPrivateKey(big.NewInt(int64(sk)))
// var pkr = regexp.MustCompile(fmt.Sprintf(
// 	`PrivateKey\{k: <redacted>, PublicKey: 0x[[:xdigit:]]{%d}\}`,
// 	2*secp256k1.CompressedPublicKeyLength))

// func TestPrintingDoesNotLeakKey(t *testing.T) {
// 	v := fmt.Sprintf("%v", k)
// 	assert.Equal(t, v+"\n", fmt.Sprintln(k))
// 	assert.Regexp(t, pkr, v)
// 	assert.NotContains(t, v, fmt.Sprintf("%x", sk))
// 	// Other verbs just give the corresponding encoding of .String()
// 	assert.Equal(t, fmt.Sprintf("%d", k), hex.EncodeToString([]byte(v)))
// }
