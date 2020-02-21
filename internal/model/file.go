package model

import (
	"time"

	"github.com/team836/clowd-storage/pkg/database"
)

type File struct {
	ID         uint      `gorm:"type:int(11) unsigned auto_increment;primary_key"`
	GoogleID   string    `gorm:"type:varchar(63);not null;unique_index:file_idx"`
	Name       string    `gorm:"type:varchar(255);not null;unique_index:file_idx"`
	Position   int16     `gorm:"type:smallint(5);not null;unique_index:file_idx"`
	Size       uint64    `gorm:"type:bigint(14) unsigned;not null"`
	UploadedAt time.Time `gorm:"type:datetime;not null;default:current_timestamp"`

	Shards []Shard `gorm:"foreignkey:FileID;association_foreignkey:ID"` // file has many shards
}

/**
Migrate file table.
*/
func MigrateFile() {
	database.
		Conn().
		Set("gorm:table_options", "CHARSET=utf8mb4").
		AutoMigrate(&File{}).
		Model(&File{}).
		AddForeignKey("google_id", "clowdees(google_id)", "RESTRICT", "CASCADE")
}
