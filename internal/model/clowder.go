package model

import (
	"time"

	"github.com/team836/clowd-storage/pkg/database"
)

type Clowder struct {
	GoogleID    string    `gorm:"type:varchar(63);primary_key"`
	MaxCapacity uint16    `gorm:"type:smallint(4) unsigned;not null;default:0"`
	SignedInAt  time.Time `gorm:"type:datetime;not null;default:current_timestamp"`
	SignedUpAt  time.Time `gorm:"type:datetime;not null;default:current_timestamp"`

	User User `gorm:"foreignkey:GoogleID;association_foreignkey:GoogleID"`
}

/**
Migrate clowder table.
*/
func MigrateClowder() {
	database.
		Conn().
		Set("gorm:table_options", "CHARSET=utf8mb4").
		AutoMigrate(&Clowder{}).
		Model(&Clowder{}).
		AddForeignKey("google_id", "users(google_id)", "CASCADE", "CASCADE")
}
