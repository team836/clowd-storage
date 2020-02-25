package model

import "github.com/team836/clowd-storage/pkg/database"

type DeletedShard struct {
	// column fields
	Name      string `gorm:"type:varchar(255);primary_key"`
	MachineID string `gorm:"type:varchar(255);not null"`
}

/**
Migrate deleted shard table.
*/
func MigrateDeletedShard() {
	database.Conn().Set("gorm:table_options", "CHARSET=utf8mb4").
		AutoMigrate(&DeletedShard{}).
		Model(&DeletedShard{}).
		AddForeignKey("machine_id", "nodes(machine_id)", "CASCADE", "CASCADE")
}
