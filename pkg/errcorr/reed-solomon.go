package errcorr

import (
	"bytes"
	"io"

	"github.com/klauspost/reedsolomon"
)

const (
	dataShards = 30

	parityShards = 20
)

/**
Encode the file using reed solomon algorithm.
*/
func Encode(file io.Reader) ([][]byte, error) {
	// make byte buffer from file
	buf := &bytes.Buffer{}
	_, _ = buf.ReadFrom(file)

	// create reed solomon encoder
	enc, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return nil, err
	}

	// split the file data
	splitData, err := enc.Split(buf.Bytes())
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
