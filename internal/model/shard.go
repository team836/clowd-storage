package model

import (
	"encoding/base64"
	"strconv"

	"github.com/team836/clowd-storage/pkg/database"
)

type Shard struct {
	// column fields
	Name            string `gorm:"type:varchar(255);primary_key"`
	Position        uint8  `gorm:"type:tinyint(3) unsigned;not null;unique_index:shard_idx"`
	FileID          uint   `gorm:"type:int(11) unsigned;not null;unique_index:shard_idx"`
	ClowderGoogleID string `gorm:"type:varchar(63);not null"`

	// associations fields
	Clowder Clowder `gorm:"foreignkey:ClowderGoogleID;association_foreignkey:GoogleID"` // shard belongs to clowder
}

/**
Migrate file table.
*/
func MigrateShard() {
	database.
		Conn().
		Set("gorm:table_options", "CHARSET=utf8mb4").
		AutoMigrate(&Shard{}).
		Model(&Shard{}).
		AddForeignKey("clowder_google_id", "clowders(google_id)", "RESTRICT", "CASCADE").
		AddForeignKey("file_id", "files(id)", "RESTRICT", "CASCADE")
}

/**
Decide shard name using combination of position and file id.
*/
func (shard *Shard) DecideName() {
	uniqueBytes := []byte(
		"p:" +
			strconv.Itoa(int(shard.Position)) +
			"f:" +
			strconv.Itoa(int(shard.FileID)),
	)

	shard.Name = base64.URLEncoding.EncodeToString(uniqueBytes)
}
