package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path"
	"runtime"
	"time"

	"github.com/team836/clowd-storage/internal/provider"

	"github.com/spf13/viper"
	"github.com/team836/clowd-storage/pkg/logger"
)

func main() {
	setMaxProcs()

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

	// open database connection
	conn := provider.DBService()
	defer conn.Close()

	// build all of the router
	router := provider.RouteService()

	// start server
	go func() {
		err := router.StartServer(&http.Server{
			Addr:           ":" + viper.GetString("APP.PORT"),
			ReadTimeout:    5 * time.Second,
			WriteTimeout:   7 * time.Second,
			MaxHeaderBytes: 1 << 20,
		})

		if err != nil {
			logger.Console().Errorf("Error starting the server, %s", err)
		}
	}()

	// wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 10 seconds
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := router.Shutdown(ctx); err != nil {
		logger.File().Error(err)
	}
}

func setMaxProcs() {
	var maxProcs int
	cpuNum := runtime.NumCPU()

	if cpuNum == 1 { // if single cpu,
		maxProcs = 2 // use 2 logical cpu for concurrency
	} else {
		maxProcs = cpuNum
	}

	runtime.GOMAXPROCS(maxProcs)
}
