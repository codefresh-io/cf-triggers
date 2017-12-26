package backend

import (
	"errors"

	"github.com/codefresh-io/hermes/pkg/model"
	"github.com/codefresh-io/hermes/pkg/util"
)

// SecretChecker checks secret or HMAC signature
type SecretChecker struct {
}

// NewSecretChecker initialize new SecretChecker
func NewSecretChecker() model.SecretChecker {
	return &SecretChecker{}
}

// Validate secret or HMAC signature
func (s *SecretChecker) Validate(message string, secret string, key string) error {
	if secret != key {
		// try to check signature
		if util.CheckHmacSignature(message, secret, key) {
			return nil
		}
		return errors.New("invalid secret or HMAC signature")
	}
	return nil
}
