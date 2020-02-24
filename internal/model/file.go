package model

import (
	"time"

	"github.com/team836/clowd-storage/pkg/database"
)

type EncFile struct {
	Model *File
	Data  [][]byte
}

type FileToLoad struct {
	Model  *File
	Shards []*ShardToLoad
}

type File struct {
	// column fields
	ID         uint      `gorm:"type:int(11) unsigned auto_increment;primary_key"`
	GoogleID   string    `gorm:"type:varchar(63);not null;unique_index:file_idx"`
	Name       string    `gorm:"type:varchar(255);not null;unique_index:file_idx"`
	Position   int16     `gorm:"type:smallint(5);not null;unique_index:file_idx"`
	Size       uint      `gorm:"type:int(11) unsigned;not null"`
	UploadedAt time.Time `gorm:"type:datetime;not null;default:current_timestamp"`
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
