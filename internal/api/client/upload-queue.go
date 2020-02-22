package client

import (
	"container/ring"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"

	"github.com/team836/clowd-storage/pkg/database"

	"github.com/team836/clowd-storage/internal/model"

	"github.com/team836/clowd-storage/internal/api/node"
)

var (
	ErrLackOfStorage = errors.New("cannot save the files because of lack of storage space")
)

type EncFile struct {
	model *model.File
	data  [][]byte
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
Assign every data shards for saving to the nodes.
And update metadata and predict nodes' status by scheduling results.

This function change nodes' status. So you SHOULD use this function with
the `NodesStatusLock` which is mutex for all nodes' status.
*/
func (uq *UploadQueue) schedule(safeRing, unsafeRing *ring.Ring) (map[*node.Node][]*node.FileOnNode, error) {
	// define constant for indicating current schedule phase
	const (
		PHASE1 = 1
		PHASE2 = 2
	)

	phase := PHASE1

	// sort the files to upload before scheduling
	uq.sort()

	currRing := safeRing
	quotas := make(map[*node.Node][]*node.FileOnNode)

	// begin a transaction
	tx := database.Conn().Begin()
	defer func() {
		// when panic is occurred, rollback all transactions
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// if cannot begin the transaction
	if err := tx.Error; err != nil {
		return nil, err
	}

	// for every files to save
	for _, file := range uq.files {
		// create the file record
		if err := tx.Create(file.model).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// for every shards
		for pos, shard := range file.data {
			tolerance := 0
			// find the node which can store this shard
			for currRing.Value.(*node.Node).Status.Capacity < uint64(len(shard)) {
				currRing = currRing.Next()

				tolerance++
				if tolerance >= currRing.Len() {
					if phase == PHASE1 {
						// change to phase2 when
						// no longer there are none possible things among the safe nodes
						phase = PHASE2
						currRing = unsafeRing
					} else if phase == PHASE2 {
						// reach at this point when
						// no longer there are none possible things among the all nodes
						tx.Rollback() // rollback the transaction
						return nil, ErrLackOfStorage
					}
				}
			}

			// set current node
			currNode := currRing.Value.(*node.Node)

			// create the shard record
			shardModel := &model.Shard{
				Position:  uint8(pos),
				FileID:    file.model.ID,
				MachineID: currNode.Model.MachineID,
				Checksum:  checksum(shard),
			}
			shardModel.DecideName()
			if err := tx.Create(shardModel).Error; err != nil {
				tx.Rollback()
				return nil, err
			}

			// assignment shard to this node
			quotas[currNode] = append(
				quotas[currNode],
				node.NewFileOnNode(shardModel.Name, shard),
			)

			// node status prediction
			currNode.Status.Capacity -= uint64(len(shard))

			currRing = currRing.Next()
		}
	}

	// commit the transaction
	tx.Commit()

	return quotas, nil
}

/**
Generate sha256 checksum about the shard.
*/
func checksum(data []byte) string {
	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:])
}
