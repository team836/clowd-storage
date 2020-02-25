package operationq

import (
	"container/ring"
	"sort"

	"github.com/team836/clowd-storage/pkg/database"

	"github.com/team836/clowd-storage/internal/model"
	"github.com/team836/clowd-storage/internal/module/spool"
)

type RestoreQueue struct {
	Shards []*model.ShardToLoad
}

func NewRQ() *RestoreQueue {
	rq := &RestoreQueue{}
	return rq
}

/**
Push the shard to queue.
*/
func (rq *RestoreQueue) Push(shards ...*model.ShardToLoad) {
	rq.Shards = append(rq.Shards, shards...)
}

/**
Assign every data shards for restoring to the nodes.
And update metadata and predict nodes' status by scheduling results.

This function change nodes' status. So you SHOULD use this function with
the `NodesStatusLock` which is mutex for all nodes' status.
*/
func (rq *RestoreQueue) Schedule(safeRing, unsafeRing *ring.Ring) (map[*spool.ActiveNode][]*model.ShardToSave, error) {
	// define constant for indicating current schedule phase
	const (
		Phase1 = 1
		Phase2 = 2
	)

	// sort the shards to restore before scheduling
	rq.sort()

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

	// for every shards
	for _, shard := range rq.Shards {
		// if curr phase is phase2, change current phase to phase1
		// and move curr ring to previous safe ring
		if phase == Phase2 {
			phase = Phase1
			prevUnsafeRing = currRing
			currRing = prevSafeRing
		}

		tolerance := 0
		// find the node which can store this shard
		for currRing.Value.(*spool.ActiveNode).Status.Capacity < uint64(len(shard.Data)) {
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

		// Before the update metadata,
		// this shard data on the previous node must be deleted when the node is reconnected.
		// Because it will be copied to another active node right now.
		// So record it to database for later deletion.
		database.Conn().
			Create(&model.DeletedShard{Name: shard.Model.Name, MachineID: shard.Model.MachineID})

		// update machine id of shard record
		if err := tx.Model(shard.Model).Update("machine_id", currNode.Model.MachineID).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// assignment shard to this node
		quotas[currNode] = append(
			quotas[currNode],
			&model.ShardToSave{
				Name: shard.Model.Name,
				Data: shard.Data,
			},
		)

		// node status prediction
		currNode.Status.Capacity -= uint64(len(shard.Data))

		currRing = currRing.Next()
	}

	// commit the transaction
	tx.Commit()

	return quotas, nil
}

func (rq *RestoreQueue) sort() {
	sort.Slice(rq.Shards, func(i, j int) bool {
		return len(rq.Shards[i].Data) > len(rq.Shards[j].Data)
	})
}
