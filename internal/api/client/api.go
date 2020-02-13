package client

import (
	"net/http"

	"github.com/team836/clowd-storage/internal/api/node"

	"github.com/labstack/echo/v4"
	"github.com/team836/clowd-storage/pkg/logger"
)

func RegisterHandlers(group *echo.Group) {
	group.POST("/file", upload)
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
		// TODO: RS algorithm
	}

	node.Pool().ClowdersStatusLock.Lock()

	node.Pool().CheckAllClowders()
	// TODO: node selection
	// TODO: clowder status prediction

	node.Pool().ClowdersStatusLock.Unlock()

	return ctx.String(http.StatusOK, "good")
}
