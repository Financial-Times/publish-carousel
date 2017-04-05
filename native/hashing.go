package native

import (
	"crypto/sha256"
	"encoding/hex"
)

// Hash hashes the given payload in SHA224 + Hex
func Hash(payload []byte) (string, error) {
	hash := sha256.New224()
	_, err := hash.Write(payload)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
