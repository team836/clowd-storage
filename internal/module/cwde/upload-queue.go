package cwde

import (
	"container/ring"
	"errors"
	"sort"

	"github.com/team836/clowd-storage/internal/module/cwdr"
	"github.com/team836/clowd-storage/pkg/database"
	"github.com/team836/clowd-storage/pkg/errcorr"

	"github.com/team836/clowd-storage/internal/model"
)

var (
	ErrLackOfStorage = errors.New("cannot save the files because of lack of storage space")
)

type EncFile struct {
	Model *model.File
	Data  [][]byte
}

type UploadQueue struct {
	files []*EncFile
}

func NewUQ() *UploadQueue {
	uq := &UploadQueue{}
	return uq
}

/**
Push the encoded file to queue.
*/
func (uq *UploadQueue) Push(file *EncFile) {
	uq.files = append(uq.files, file)
}

/**
Assign every data shards for saving to the nodes.
And update metadata and predict nodes' status by scheduling results.

This function change nodes' status. So you SHOULD use this function with
the `NodesStatusLock` which is mutex for all nodes' status.
*/
func (uq *UploadQueue) Schedule(safeRing, unsafeRing *ring.Ring) (map[*cwdr.ActiveNode][]*model.ShardToSave, error) {
	// define constant for indicating current schedule phase
	const (
		PHASE1 = 1
		PHASE2 = 2
	)

	phase := PHASE1

	// sort the files to upload before scheduling
	uq.sort()

	currRing := safeRing
	quotas := make(map[*cwdr.ActiveNode][]*model.ShardToSave)

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
		if err := tx.Create(file.Model).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// for every shards
		for pos, shard := range file.Data {
			tolerance := 0
			// find the node which can store this shard
			for currRing.Value.(*cwdr.ActiveNode).Status.Capacity < uint64(len(shard)) {
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
			currNode := currRing.Value.(*cwdr.ActiveNode)

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
	sort.Slice(uq.files, func(i, j int) bool {
		return len(uq.files[i].Data[0]) > len(uq.files[j].Data[0])
	})
}
