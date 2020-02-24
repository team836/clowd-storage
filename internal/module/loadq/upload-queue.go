package loadq

import (
	"container/ring"
	"errors"
	"sort"

	"github.com/team836/clowd-storage/internal/module/spool"
	"github.com/team836/clowd-storage/pkg/database"
	"github.com/team836/clowd-storage/pkg/errcorr"

	"github.com/team836/clowd-storage/internal/model"
)

var (
	ErrLackOfStorage = errors.New("cannot save the files because of lack of storage space")
)

type UploadQueue struct {
	Files []*model.EncFile
}

func NewUQ() *UploadQueue {
	uq := &UploadQueue{}
	return uq
}

/**
Push the encoded file to queue.
*/
func (uq *UploadQueue) Push(file *model.EncFile) {
	uq.Files = append(uq.Files, file)
}

/**
Assign every data shards for saving to the nodes.
And update metadata and predict nodes' status by scheduling results.

This function change nodes' status. So you SHOULD use this function with
the `NodesStatusLock` which is mutex for all nodes' status.
*/
func (uq *UploadQueue) Schedule(safeRing, unsafeRing *ring.Ring) (map[*spool.ActiveNode][]*model.ShardToSave, error) {
	// define constant for indicating current schedule phase
	const (
		Phase1 = 1
		Phase2 = 2
	)

	// sort the files to upload before scheduling
	uq.sort()

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

	phase := Phase1
	prevSafeRing := safeRing
	prevUnsafeRing := unsafeRing
	currRing := safeRing
	quotas := make(map[*spool.ActiveNode][]*model.ShardToSave)

	// for every files to save
	for _, file := range uq.Files {
		// create the file record
		if err := tx.Create(file.Model).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// if curr phase is phase2, change current phase to phase1
		// and move curr ring to previous safe ring
		if phase == Phase2 {
			phase = Phase1
			prevUnsafeRing = currRing
			currRing = prevSafeRing
		}

		// for every shards
		for pos, shard := range file.Data {
			tolerance := 0
			// find the node which can store this shard
			for currRing.Value.(*spool.ActiveNode).Status.Capacity < uint64(len(shard)) {
				currRing = currRing.Next()

				tolerance++
				if tolerance >= currRing.Len() {
					if phase == Phase1 {
						// change to phase2 when
						// no longer there are none possible things among the safe nodes
						phase = Phase2
						prevSafeRing = currRing   // save current safe ring
						currRing = prevUnsafeRing // move curr ring to previous unsafe ring
						tolerance = 0
					} else if phase == Phase2 {
						// reach at this point when
						// no longer there are none possible things among the all nodes
						tx.Rollback() // rollback the transaction
						return nil, ErrLackOfStorage
					}
				}
			}

			// set current node
			currNode := currRing.Value.(*spool.ActiveNode)

			// create the shard record
			shardModel := &model.Shard{
				Position:  uint8(pos),
				FileID:    file.Model.ID,
				MachineID: currNode.Model.MachineID,
				Checksum:  errcorr.Checksum(shard),
			}
			shardModel.DecideName()
			if err := tx.Create(shardModel).Error; err != nil {
				tx.Rollback()
				return nil, err
			}

			// assignment shard to this node
			quotas[currNode] = append(
				quotas[currNode],
				&model.ShardToSave{
					Name: shardModel.Name,
					Data: shard,
				},
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
Sort the files by shard size in descending order.
*/
func (uq *UploadQueue) sort() {
	sort.Slice(uq.Files, func(i, j int) bool {
		return len(uq.Files[i].Data[0]) > len(uq.Files[j].Data[0])
	})
}
