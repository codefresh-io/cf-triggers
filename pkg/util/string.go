package util

import (
	"encoding/hex"
)

// IsHexString check if string is a valid Hex string
func IsHexString(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}
