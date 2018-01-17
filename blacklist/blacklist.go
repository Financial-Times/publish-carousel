package blacklist

import (
	"bufio"
	"bytes"
	"os"
)

var NoOpBlacklist = func(uuid string) (bool, error) { return false, nil }

// IsBlacklisted filter function
type IsBlacklisted func(uuid string) (bool, error)

// NewFileBasedBlacklist returns a function of type Blacklist which will cross-check uuids against a file based blacklist
func NewFileBasedBlacklist(file string) (IsBlacklisted, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return func(uuid string) (bool, error) {
		f, err := os.Open(file)
		if err != nil {
			return false, err
		}
		defer f.Close()

		uuidAsBytes := []byte(uuid)

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if bytes.Contains(scanner.Bytes(), uuidAsBytes) {
				return true, nil
			}
		}

		err = scanner.Err()
		return false, err
	}, nil
}
