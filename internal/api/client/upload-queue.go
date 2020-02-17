package client

import "sort"

type EncFile struct {
	fileID string
	data   [][]byte
}

type UploadQueue struct {
	files []*EncFile
}

func newUQ() *UploadQueue {
	uq := &UploadQueue{}
	return uq
}

/**
Push the encoded file to queue.
*/
func (uq *UploadQueue) push(file *EncFile) {
	uq.files = append(uq.files, file)
}

/**
Sort the files by shard size in descending order.
*/
func (uq *UploadQueue) sort() {
	sort.Slice(uq.files, func(i, j int) bool {
		return len(uq.files[i].data[0]) > len(uq.files[j].data[0])
	})
}
