package backend

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
)

// check hmac signature with: sha1, sha256, sha512
// signature is a hex encoded string
func checkSignature(message, signature, secret string) bool {
	hashes := [](func() hash.Hash){sha1.New, sha256.New, sha512.New}
	for _, h := range hashes {
		hash := hmac.New(h, []byte(secret))
		hash.Write([]byte(message))
		expectedMAC := hash.Sum(nil)
		sig, _ := hex.DecodeString(signature)
		if hmac.Equal(sig, expectedMAC) {
			return true
		}
	}
	return false
}
