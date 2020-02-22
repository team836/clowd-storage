package model

import (
	"time"

	"github.com/team836/clowd-storage/pkg/database"
)

type Clowder struct {
	// column fields
	GoogleID   string    `gorm:"type:varchar(63);primary_key"`
	SignedInAt time.Time `gorm:"type:datetime;not null;default:current_timestamp"`
	SignedUpAt time.Time `gorm:"type:datetime;not null;default:current_timestamp"`

	// associations fields
	Nodes []Node `gorm:"foreignkey:ClowderGoogleID;association_foreignkey:GoogleID"` // clowder has many nodes
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
