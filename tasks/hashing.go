package tasks

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/Financial-Times/nativerw/logging"
)

// Hash hashes the given payload in SHA224 + Hex
func Hash(payload string) string {
	hash := sha256.New224()
	_, err := hash.Write([]byte(payload))
	if err != nil {
		logging.Warn("Failed to write hash!")
	}

	return hex.EncodeToString(hash.Sum(nil))
}
