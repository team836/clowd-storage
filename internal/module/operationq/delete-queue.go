package operationq

import (
	"github.com/team836/clowd-storage/internal/model"
	"github.com/team836/clowd-storage/pkg/database"
	"github.com/team836/clowd-storage/pkg/logger"
)

type DeleteQueue struct {
	Files []*model.File
}

func NewDelQ() *DeleteQueue {
	delQ := &DeleteQueue{}
	return delQ
}

/**
Push the files to delete.
*/
func (delQ *DeleteQueue) Push(googleID string, fileNames ...string) error {
	files := make([]*model.File, 0)

	// find all segments of the file using google id and name
	sqlResult := database.Conn().
		Where("google_id = ? AND name IN (?)", googleID, fileNames).
		Preload("Shards").
		Find(&files)

	if sqlResult.Error != nil {
		// if the file is not exist in the record
		if sqlResult.RecordNotFound() {
			return ErrFileNotExist
		}

		// other sql error
		logger.File().Errorf("Error finding the file in database, %s", sqlResult.Error.Error())
		return sqlResult.Error
	}

	delQ.Files = files

	return nil
}
