package client

import (
	"container/ring"
	"errors"
	"sort"

	"github.com/team836/clowd-storage/internal/api/node"
)

type FileHeader struct {
	name  string
	order int
	size  uint
}

type EncFile struct {
	header *FileHeader
	data   [][]byte
}

type UploadQueue struct {
	clowdee *model.Clowdee
	files   []*EncFile
}

func newUQ(clowdee *model.Clowdee) *UploadQueue {
	uq := &UploadQueue{clowdee: clowdee}
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
func (uq *UploadQueue) schedule(safeRing, unsafeRing *ring.Ring) (map[*node.Clowder][]*node.FileOnNode, error) {
	// define constant for indicating current schedule phase
	const (
		PHASE1 = 1
		PHASE2 = 2
	)

	phase := PHASE1

	// sort the files to upload before scheduling
	uq.sort()

	currRing := safeRing
	quotas := make(map[*node.Clowder][]*node.FileOnNode)
	// for every files to save
	for _, file := range uq.files {
		// for every shards
		for _, shard := range file.data {
			tolerance := 0
			// find the clowder which can store this shard
			for currRing.Value.(*node.Clowder).Status.Capacity < uint64(len(shard)) {
				currRing = currRing.Next()

				tolerance++
				if tolerance >= currRing.Len() {
					if phase == PHASE1 {
						// change to phase2 when
						// no longer there are none possible things among the safe clowders
						phase = PHASE2
						currRing = unsafeRing
					} else if phase == PHASE2 {
						// reach at this point when
						// no longer there are none possible things among the all clowders
						return nil, errors.New("cannot save the files because of lack of storage space")
					}
				}
			}

			currClowder := currRing.Value.(*node.Clowder)

			// assignment shard to this clowder
			quotas[currClowder] = append(
				quotas[currClowder],
				node.NewFileOnNode("0", shard), // TODO: set valid file name
			)

			// TODO: save metadata to the database using goroutine

			// clowder status prediction
			currClowder.Status.Capacity -= uint64(len(shard))

			currRing = currRing.Next()
		}
	}

	return quotas, nil
}
