package errcorr

import (
	"bytes"
	"encoding/base64"

	"github.com/klauspost/reedsolomon"
)

/**
Expansion factor of shards determine durability of recovery.

Exp. factor = (count of parity shards) / (count of data shards)

Currently our percentage of recovery is 99.970766304935266% given 10% failure.
(See https://storj.io/storjv3.pdf document)
*/
const (
	// count of data shards
	dataShards = 30

	// count of parity shards
	parityShards = 20
)

/**
Encode the file data using reed solomon algorithm.
*/
func Encode(base64Data string) ([][]byte, uint, error) {
	// convert base64 data to byte array
	bytes, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, 0, err
	}

	// create reed solomon encoder
	enc, _ := reedsolomon.New(dataShards, parityShards)

	// split the file data
	splitData, err := enc.Split(bytes)
	if err != nil {
		return nil, 0, err
	}

	// encode the split file using reed solomon algorithm
	err = enc.Encode(splitData)
	if err != nil {
		return nil, 0, err
	}

	return splitData, uint(len(bytes)), nil
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

	// convert byte array to base64 string
	data := base64.StdEncoding.EncodeToString(buf.Bytes())

	return data, nil
}
