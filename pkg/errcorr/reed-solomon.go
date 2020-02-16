package errcorr

import (
	"bytes"
	"io"

	"github.com/team836/clowd-storage/pkg/logger"

	"github.com/klauspost/reedsolomon"
)

const (
	dataShards = 100

	parityShards = 50
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
		logger.File().Infof("Error creating new encoder for reed solomon, %s", err)
		return nil, err
	}

	// split the file data
	splitData, err := enc.Split(buf.Bytes())
	if err != nil {
		logger.File().Infof("Error splitting the file data, %s", err)
		return nil, err
	}

	// encode the split file using reed solomon algorithm
	err = enc.Encode(splitData)
	if err != nil {
		logger.File().Infof("Error encoding the file, %s", err)
		return nil, err
	}

	return splitData, nil
}
