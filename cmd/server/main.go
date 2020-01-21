package main

import (
	"os"
	"path"

	"github.com/team836/clowd-storage/pkg/logger"

	"github.com/spf13/viper"
)

func main() {
	currDir, err := os.Getwd()
	if err != nil {
		logger.Console().Fatal(err)
	}

	// set app root path as config
	viper.Set("AppRoot", path.Join(currDir, "../../"))

	// load env file
	viper.SetConfigFile(path.Join(viper.GetString("AppRoot"), "./env.yml"))
	if err := viper.ReadInConfig(); err != nil {
		logger.Console().Fatalf("Error reading env file, %s", err)
	}
}
