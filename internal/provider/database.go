package provider

import (
	"github.com/jinzhu/gorm"
	"github.com/team836/clowd-storage/pkg/database"
)

func DBService() *gorm.DB {
	conn := database.Conn()

	return conn
}
