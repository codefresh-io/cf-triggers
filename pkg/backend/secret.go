package backend

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"hash"

	"github.com/codefresh-io/hermes/pkg/model"
	"fmt"
)

// SecretChecker checks secret or HMAC signature
type SecretChecker struct {
}

// NewSecretChecker initialize new SecretChecker
func NewSecretChecker() model.SecretChecker {
	return &SecretChecker{}
}

// helper function to check HMAC sha1/256/512 signatures
func checkHmacSignature(message, signature, secret string) bool {
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

// Validate secret or HMAC signature
func (s *SecretChecker) Validate(message string, secret string, key string) error {
	fmt.Println(message + " " + secret + " " + key)
	if secret != key {
		// try to check signature
		if checkHmacSignature(message, secret, key) {
			return nil
		}
		return errors.New("invalid secret or HMAC signature")
	}
	return nil
}
