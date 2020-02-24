package loadq

import (
	"github.com/team836/clowd-storage/internal/model"
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
