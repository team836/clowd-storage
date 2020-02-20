package model

import (
	"time"

	"github.com/team836/clowd-storage/pkg/database"
)

type File struct {
	GoogleID   string    `gorm:"type:varchar(63);primary_key"`
	Name       string    `gorm:"type:varchar(255);primary_key"`
	Order      int16     `gorm:"type:smallint(5);primary_key"`
	Size       uint64    `gorm:"type:bigint(14) unsigned;not null"`
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
