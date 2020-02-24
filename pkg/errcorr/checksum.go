package errcorr

import (
	"crypto/sha256"
	"encoding/hex"
)

/**
Generate sha256 checksum of the shard.
*/
func Checksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

/**
Check whether if the checksum of specific data is valid
*/
func IsCorruptedChecksum(data []byte, checksum string) bool {
	return Checksum(data) != checksum
}
