package model

import (
	"crypto/md5"
	"encoding/hex"
	"strconv"

	"github.com/team836/clowd-storage/pkg/database"
)

type Shard struct {
	// column fields
	Name      string `gorm:"type:varchar(255);primary_key"`
	Position  uint8  `gorm:"type:tinyint(3) unsigned;not null;unique_index:shard_idx"`
	FileID    uint   `gorm:"type:int(11) unsigned;not null;unique_index:shard_idx"`
	MachineID string `gorm:"type:varchar(255);not null"`
	Checksum  string `gorm:"type:char(64);not null"`
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
		AddForeignKey("file_id", "files(id)", "RESTRICT", "CASCADE").
		AddForeignKey("machine_id", "nodes(machine_id)", "RESTRICT", "CASCADE")
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

	hash := md5.Sum(uniqueBytes)
	shard.Name = hex.EncodeToString(hash[:])
}
