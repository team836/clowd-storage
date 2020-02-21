package provider

import (
	"github.com/jinzhu/gorm"
	"github.com/team836/clowd-storage/internal/model"
	"github.com/team836/clowd-storage/pkg/database"
)

/**
Boot database service.
*/
func DBService() *gorm.DB {
	conn := database.Conn()
	model.MigrateUser()
	model.MigrateClowdee()
	model.MigrateClowder()
	model.MigrateFile()
	model.MigrateShard()

	return conn
}
