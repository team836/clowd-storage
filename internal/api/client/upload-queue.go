package client

import (
	"container/ring"
	"sort"

	"github.com/team836/clowd-storage/internal/api/node"
)

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

/**
Assign every data shards for saving to the clowders.
And update metadata and predict clowders' status by scheduling results.

This function change clowders' status. So you SHOULD use this function with
the `ClowdersStatusLock` which is mutex for all clowders' status.
*/
func (uq *UploadQueue) schedule(clowders *ring.Ring) map[*node.Clowder][]*node.FileOnNode {
	// sort the files to upload before scheduling
	uq.sort()

	quotas := make(map[*node.Clowder][]*node.FileOnNode)
	// for every files to save
	for _, file := range uq.files {
		// for every shards
		for _, shard := range file.data {
			currClowder := clowders.Value.(*node.Clowder)

			// assignment shard to this clowder
			quotas[currClowder] = append(
				quotas[currClowder],
				node.NewFileOnNode("0", shard), // TODO: set valid file name
			)

			// TODO: save metadata to the database using goroutine

			// clowder status prediction
			currClowder.Status.Capacity -= uint64(len(shard))

			clowders = clowders.Next()
		}
	}

	return quotas
}
