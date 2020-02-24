package model

import "github.com/team836/clowd-storage/pkg/database"

type User struct {
	// column fields
	GoogleID string `gorm:"type:varchar(63);primary_key"`
	Email    string `gorm:"type:varchar(255);not null;unique"`
	Name     string `gorm:"type:varchar(63);not null"`
	Image    string `gorm:"type:varchar(255);not null"`
}

/**
Migrate user table.
*/
func MigrateUser() {
	database.
		Conn().
		Set("gorm:table_options", "CHARSET=utf8mb4").
		AutoMigrate(&User{})
}
