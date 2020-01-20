package main

import (
	"log"
	"os"
	"path"

	"github.com/spf13/viper"
)

func main() {
	currDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// set app root path as config
	viper.Set("AppRoot", path.Join(currDir, "../../"))

	// load env file
	viper.SetConfigFile(path.Join(viper.GetString("AppRoot"), "./env.yml"))
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading env file, %s", err)
	}
}
