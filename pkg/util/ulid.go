package util

import (
	"github.com/oklog/ulid"
	"math/rand"
	"time"
)

// GenerateULID generate Unique Lexicographically Sortable Identifier
func GenerateULID() (string, error) {
	seed := time.Now().UnixNano()
	source := rand.NewSource(seed)
	entropy := rand.New(source)
	ulid, err := ulid.New(ulid.Timestamp(time.Now()), entropy)
	return ulid.String(), err
}
