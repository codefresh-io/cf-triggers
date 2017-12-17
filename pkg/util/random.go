package util

import (
	"math/rand"
	"time"
)

var r *rand.Rand // Rand for this package.

// TestMode test mode
var TestMode = false

// TestRandomString random string for test
const TestRandomString = "pQ8GqKTHv8vgWrtW"

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// RandomString generates random string
func RandomString(strlen int) string {
	// use contant string for test mode
	if TestMode {
		return TestRandomString
	}

	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := ""
	for i := 0; i < strlen; i++ {
		index := r.Intn(len(chars))
		result += chars[index : index+1]
	}
	return result
}
