package provider

import (
	"github.com/jinzhu/gorm"
	"github.com/team836/clowd-storage/pkg/database"
)

/**
Boot database service.
*/
func DBService() *gorm.DB {
	conn := database.Conn()

	return conn
}
