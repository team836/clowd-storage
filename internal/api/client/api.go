package client

import (
	"net/http"

	"github.com/team836/clowd-storage/pkg/errcorr"

	"github.com/team836/clowd-storage/pkg/logger"

	"github.com/team836/clowd-storage/internal/api/node"

	"github.com/labstack/echo/v4"
)

func RegisterHandlers(group *echo.Group) {
	group.POST("/file", upload)
}

/**
File upload requested by client(clowdee).

- Get uploading file.
- Run Reed-Solomon algorithm to each files.
- Check all nodes(clowders)' status by ping and pong.
- By all nodes' status, run the node selection.
- Update the selected nodes' status by prediction.
- Finally save the files to the nodes.
*/
func upload(ctx echo.Context) error {
	form, err := ctx.MultipartForm()
	if err != nil {
		logger.File().Infof("Error uploading client's file, %s", err)
		return nil
	}
	fileHeaders := form.File["files"]

	for _, fileHeader := range fileHeaders {
		file, err := fileHeader.Open()
		if err != nil {
			logger.File().Infof("Error opening uploaded file, %s", err)
			return nil
		}

		shards, err := errcorr.Encode(file)
		if err != nil {
			logger.File().Infof("Error encoding the file, %s", err)
			return nil
		}
	}

	node.Pool().ClowdersStatusLock.Lock()

	node.Pool().CheckAllClowders()
	// TODO: node selection
	// TODO: clowder status prediction

	node.Pool().ClowdersStatusLock.Unlock()

	return ctx.String(http.StatusOK, "good")
}
