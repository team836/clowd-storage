package client

import (
	"errors"

	"github.com/team836/clowd-storage/internal/model"
	"github.com/team836/clowd-storage/pkg/database"
	"github.com/team836/clowd-storage/pkg/logger"
)

var (
	ErrFileNotExist = errors.New("file is not exists")
)

type DownloadQueue struct {
	files []*model.File
}

type ShardToLoad struct {
	model *model.Shard
	data  []byte
}

func newDQ() *DownloadQueue {
	dq := &DownloadQueue{}
	return dq
}

/**
Push the list of file to load by querying database.
*/
func (dq *DownloadQueue) push(fileToLoad *model.File) error {
	fileModels := &[]*model.File{}

	// find all segments of the file
	// and preload its corresponding shards
	sqlResult := database.Conn().
		Where(fileToLoad).
		Preload("Shards").Find(fileModels)

	if sqlResult.Error != nil {
		// if the file is not exist in the record
		if sqlResult.RecordNotFound() {
			return ErrFileNotExist
		}

		// other sql error
		logger.File().Errorf("Error finding the file in database, %s", sqlResult.Error.Error())
		return sqlResult.Error
	}

	// append to download queue
	dq.files = append(dq.files, *fileModels...)

	return nil
}

/**
Assign shards for download to the each nodes
which are identified by machine id.
*/
func (dq *DownloadQueue) schedule() map[string][]*ShardToLoad {
	quotas := make(map[string][]*ShardToLoad)

	// for every files to download
	for _, file := range dq.files {
		// for every shards of the file
		for _, shard := range file.Shards {
			shard := shard // pin (See the scopelint document)

			quotas[shard.MachineID] = append(
				quotas[shard.MachineID],
				&ShardToLoad{
					model: &shard,
				},
			)
		}
	}

	return quotas
}
