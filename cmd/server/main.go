package main

import (
	"os"
	"path"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"github.com/team836/clowd-storage/pkg/logger"
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

	router := buildRouter()
}

func buildRouter() *echo.Echo {
	// open the access log file
	accessLogFile, err := os.OpenFile(
		path.Join(viper.GetString("AppRoot"), "/logs/access.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)

	// if cannot open the file, using default stdout
	if err != nil {
		logger.Console().Warn("Failed to logging to file(`access.log`), using default stdout")
		accessLogFile = os.Stdout
	}

	// basic middleware list
	basicMiddleware := []echo.MiddlewareFunc{
		middleware.BodyLimit("200MB"),
		middleware.CORS(),
		middleware.Gzip(),
		middleware.LoggerWithConfig(middleware.LoggerConfig{
			Format: `{"time":"${time_rfc3339_nano}","remote_ip":"${remote_ip}","host":"${host}",` +
				`"method":"${method}","uri":"${uri}","status":${status},"error":"${error}",` +
				`"latency":${latency},"bytes_in":${bytes_in},"bytes_out":${bytes_out}}` + "\n",
			Output: accessLogFile,
		}),
		middleware.Recover(),
		middleware.Secure(),
	}

	mainRouter := echo.New()
	mainRouter.Use(basicMiddleware...)
	v1Group := mainRouter.Group("/v1")
	api.RegisterHandlers(v1Group)

	return mainRouter
}
