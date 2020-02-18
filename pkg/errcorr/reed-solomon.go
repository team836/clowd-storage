package errcorr

import (
	"encoding/base64"

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
