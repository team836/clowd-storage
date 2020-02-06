package database

import (
	"sync"

	"github.com/team836/clowd-storage/pkg/logger"

	"github.com/spf13/viper"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql" // import database driver
)

var (
	conn *gorm.DB  // singleton instance
	once sync.Once // for thread safe singleton
)

/**
Return the singleton database connection instance.
*/
func Conn() *gorm.DB {
	once.Do(func() {
		conn = newConnection()
	})

	return conn
}

func newConnection() *gorm.DB {
	// open connection to mysql
	conn, err := gorm.Open(
		"mysql",
		viper.GetString("DB.USER")+
			":"+
			viper.GetString("DB.PASSWORD")+
			"@("+
			viper.GetString("DB.HOST")+
			")/"+
			viper.GetString("DB.DBNAME")+
			"?charset=utf8&parseTime=True&loc=Local",
	)

	if err != nil {
		logger.Console().Fatalf("Error connecting to database, %s", err)
	}

	// config for connection pool
	conn.DB().SetMaxIdleConns(10)
	conn.DB().SetMaxOpenConns(100)

	return conn
}
