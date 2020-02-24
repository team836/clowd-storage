package model

import (
	"time"

	"github.com/team836/clowd-storage/pkg/database"
)

type Clowdee struct {
	// column fields
	GoogleID   string    `gorm:"type:varchar(63);primary_key"`
	SignedInAt time.Time `gorm:"type:datetime;not null;default:current_timestamp"`
	SignedUpAt time.Time `gorm:"type:datetime;not null;default:current_timestamp"`

	// associations fields
	Files []File `gorm:"foreignkey:GoogleID;association_foreignkey:GoogleID"` // clowdee has many files
}

/**
Migrate clowdee table.
*/
func MigrateClowdee() {
	database.
		Conn().
		Set("gorm:table_options", "CHARSET=utf8mb4").
		AutoMigrate(&Clowdee{}).
		Model(&Clowdee{}).
		AddForeignKey("google_id", "users(google_id)", "CASCADE", "CASCADE")
}
