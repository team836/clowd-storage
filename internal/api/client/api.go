package client

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/team836/clowd-storage/pkg/logger"
)

func RegisterHandlers(group *echo.Group) {
	group.POST("/file", upload)
}

/**
File upload request for checking whether there are available nodes to store.
*/
func uploadRequest(ctx echo.Context) error {
	return nil
}

/**
File upload
*/
func upload(ctx echo.Context) error {
	form, err := ctx.MultipartForm()
	if err != nil {
		logger.File().Infof("Error uploading client's file, %s", err)
		return nil
	}

	files := form.File["files"]
	for _, file := range files {
		logger.Console().Infof("file name: %s", file.Filename)
	}

	return ctx.String(http.StatusOK, "good")
}
