package cwde

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
	Files []*model.FileToLoad
}

func NewDQ() *DownloadQueue {
	dq := &DownloadQueue{}
	return dq
}

/**
Push the list of file to load by querying database.
*/
func (dq *DownloadQueue) Push(googleID string, fileName string) error {
	fileModels := &[]*model.File{}

	// find all segments of the file using google id and name
	sqlResult := database.Conn().
		Where(&model.File{GoogleID: googleID, Name: fileName}).
		Find(fileModels)

	if sqlResult.Error != nil {
		// if the file is not exist in the record
		if sqlResult.RecordNotFound() {
			return ErrFileNotExist
		}

		// other sql error
		logger.File().Errorf("Error finding the file in database, %s", sqlResult.Error.Error())
		return sqlResult.Error
	}

	// for every file records(segments)
	for _, fileModel := range *fileModels {
		fileToLoad := &model.FileToLoad{Model: fileModel}

		// find all shards of the segment which are ordered by its position
		shardModels := &[]*model.Shard{}
		sqlResult := database.Conn().
			Where("file_id = ?", fileModel.ID).
			Order("position asc").
			Find(shardModels)

		if sqlResult.Error != nil {
			// if the shard which is corresponding to the file is not exist in the record
			if sqlResult.RecordNotFound() {
				return ErrFileNotExist
			}

			// other sql error
			logger.File().Errorf("Error finding the shard in database, %s", sqlResult.Error.Error())
			return sqlResult.Error
		}

		// for every shards
		for _, shardModel := range *shardModels {
			shardToLoad := &model.ShardToLoad{Model: shardModel}
			fileToLoad.Shards = append(fileToLoad.Shards, shardToLoad)
		}

		dq.Files = append(dq.Files, fileToLoad)
	}

	return nil
}

/**
Assign shards for download to the each nodes
which are identified by machine id.
*/
func (dq *DownloadQueue) Schedule() map[string][]*model.ShardToLoad {
	quotas := make(map[string][]*model.ShardToLoad)

	// for every files to download
	for _, file := range dq.Files {
		// for every shards of the file
		for _, shard := range file.Shards {
			quotas[shard.Model.MachineID] = append(
				quotas[shard.Model.MachineID],
				shard,
			)
		}
	}

	return quotas
}
