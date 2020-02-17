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

- Get uploaded file.
- Run Reed-Solomon algorithm to each files.
- Check all nodes(clowders)' status by ping and pong.
- By all nodes' status, run the node selection.
- Schedule saving for every shards.
- Update the selected nodes' status by prediction.
- Finally save the files to the nodes.
*/
func upload(ctx echo.Context) error {
	form, err := ctx.MultipartForm()
	if err != nil {
		logger.File().Infof("Error uploading client's file, %s", err)
		return nil
	}

	uq := newUQ()
	fileHeaders := form.File["files"]
	for _, fileHeader := range fileHeaders {
		// open file
		file, err := fileHeader.Open()
		if err != nil {
			logger.File().Infof("Error opening uploaded file, %s", err)
			return nil
		}

		// encode file using reed solomon algorithm
		shards, err := errcorr.Encode(file)
		if err != nil {
			logger.File().Infof("Error encoding the file, %s", err)
			return nil
		}

		encFile := &EncFile{fileID: "0", data: shards}
		uq.push(encFile)
	}

	// this area almost change all clowders' status
	// so, protect it using mutex for all clowder's status
	node.Pool().ClowdersStatusLock.Lock()

	// check all clowders' status by ping&pong
	node.Pool().CheckAllClowders()

	// node selection
	clowders := node.Pool().SelectClowders()
	if clowders.Len() == 0 {
		logger.File().Errorf("Available clowders are not exist.")
		return nil
	}

	// schedule saving for every shards to the clowders
	// and get results
	quotas := uq.schedule(clowders)

	// end of mutex area
	node.Pool().ClowdersStatusLock.Unlock()

	// save each quota using goroutine
	for clowder, file := range quotas {
		go func(c *node.Clowder, f []*node.FileOnNode) {
			c.SaveFile <- f
		}(clowder, file)
	}

	return ctx.NoContent(http.StatusCreated)
}
