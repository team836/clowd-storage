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

func (uq *UploadQueue) push(file *EncFile) {
	uq.files = append(uq.files, file)
}

func (uq *UploadQueue) pop() *EncFile {
	var file *EncFile
	file, uq.files = uq.files[0], uq.files[1:]
	return file
}

func (uq *UploadQueue) sort() {
	sort.Slice(uq.files, func(i, j int) bool {
		return len(uq.files[i].data[0]) > len(uq.files[j].data[0])
	})
}
