package errcorr

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"github.com/klauspost/reedsolomon"
)

const (
	dataShards = 30

	parityShards = 20
)

/**
Encode the file data using reed solomon algorithm.
*/
func Encode(data string) ([][]byte, error) {
	// make byte buffer from base64 string
	bytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	// create reed solomon encoder
	enc, _ := reedsolomon.New(dataShards, parityShards)

	// split the file data
	splitData, err := enc.Split(bytes)
	if err != nil {
		return nil, err
	}

	// encode the split file using reed solomon algorithm
	err = enc.Encode(splitData)
	if err != nil {
		return nil, err
	}

	return splitData, nil
}

/**
Decode the shards to the original file using reed solomon algorithm.
When some data are missed, reconstruct them.
*/
func Decode(shards [][]byte, dataSize int) (string, error) {
	// create read solomon encoder
	enc, _ := reedsolomon.New(dataShards, parityShards)

	// decode(reconstruct) the missing shards
	err := enc.Reconstruct(shards)
	if err != nil {
		return "", err
	}

	// join the shards
	buf := &bytes.Buffer{}
	err = enc.Join(buf, shards, dataSize)
	if err != nil {
		return "", err
	}

	// make base64 string from byte buffer
	data := base64.StdEncoding.EncodeToString(buf.Bytes())

	return data, nil
}

/**
Generate sha256 checksum about the shard.
*/
func Checksum(data []byte) string {
	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:])
}
