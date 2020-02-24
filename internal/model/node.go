package model

import "github.com/team836/clowd-storage/pkg/database"

type Node struct {
	// column fields
	MachineID       string `gorm:"type:varchar(255);primary_key"`
	MaxCapacity     uint16 `gorm:"type:smallint(4) unsigned;not null;default:1"`
	ClowderGoogleID string `gorm:"type:varchar(63);not null"`

	// associations fields
	Shards []Shard `gorm:"foreignkey:MachineID;association_foreignkey:MachineID"` // node has many shards
}

/**
Migrate node table.
*/
func MigrateNode() {
	database.
		Conn().
		Set("gorm:table_options", "CHARSET=utf8mb4").
		AutoMigrate(&Node{}).
		Model(&Node{}).
		AddForeignKey("clowder_google_id", "clowders(google_id)", "CASCADE", "CASCADE")
}
