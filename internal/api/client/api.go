package client

import (
	"net/http"
	"time"

	"github.com/team836/clowd-storage/pkg/database"

	"github.com/team836/clowd-storage/internal/model"

	"github.com/team836/clowd-storage/pkg/errcorr"

	"github.com/team836/clowd-storage/pkg/logger"

	"github.com/team836/clowd-storage/internal/api/node"

	"github.com/labstack/echo/v4"
)

type FileOnClient struct {
	Name  string `json:"name"`
	Order int    `json:"order"`
	Data  string `json:"data"` // base64 encoded
}

type FileView struct {
	Name       string    `json:"name"`
	Size       uint      `json:"size"`
	UploadedAt time.Time `json:"uploadedAt"`
}

type FileToDownload struct {
	Name string `json:"name"`
}

func RegisterHandlers(group *echo.Group) {
	group.GET("/dir", fileList)
	group.POST("/files", upload)
	group.GET("/files", download)
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
	clowdee := ctx.Get("clowdee").(*model.Clowdee)

	// bind uploaded data into array of `FileOnClient` struct
	files := &[]*FileOnClient{}
	if err := ctx.Bind(files); err != nil {
		logger.File().Infof("Error binding client's uploaded file, %s", err)
		return err
	}

	// create upload queue
	uq := newUQ(clowdee)

	// encode every file data using reed solomon algorithm
	for _, file := range *files {
		shards, err := errcorr.Encode(file.Data)
		if err != nil {
			logger.File().Infof("Error encoding the file, %s", err)
			return ctx.String(http.StatusNotAcceptable, "Cannot handle this file: "+file.Name)
		}

		encFile := &EncFile{
			header: &FileHeader{
				name:  file.Name,
				order: file.Order,
				size:  uint(len(file.Data)),
			},
			data: shards,
		}
		uq.push(encFile)
	}

	// this area almost change all clowders' status
	// so, protect it using mutex for all clowder's status
	node.Pool().ClowdersStatusLock.Lock()
	defer func() {
		// when panic is occurred, unlock the clowders status lock
		if r := recover(); r != nil {
			node.Pool().ClowdersStatusLock.Unlock()
		}
	}()

	// check all clowders' status by ping&pong
	node.Pool().CheckAllClowders()

	// node selection
	safeRing, unsafeRing := node.Pool().SelectClowders()
	if safeRing.Len()+unsafeRing.Len() == 0 {
		node.Pool().ClowdersStatusLock.Unlock()
		logger.File().Errorf("Available clowders are not exist.")
		return ctx.String(http.StatusNotAcceptable, "Cannot save the files because currently there are no available clowders")
	}

	// schedule saving for every shards to the clowders
	// and get results
	quotas, err := uq.schedule(safeRing, unsafeRing)
	if err != nil {
		node.Pool().ClowdersStatusLock.Unlock()
		logger.File().Errorf("Error scheduling upload, %s", err)

		if err == ErrLackOfStorage {
			return ctx.String(http.StatusNotAcceptable, err.Error())
		}

		return ctx.NoContent(http.StatusInternalServerError)
	}

	// save each quota using goroutine
	for clowder, file := range quotas {
		go func(c *node.Clowder, f []*node.FileOnNode) {
			c.SaveFile <- f
		}(clowder, file)
	}

	// end of mutex area for clowders status lock
	node.Pool().ClowdersStatusLock.Unlock()

	return ctx.NoContent(http.StatusCreated)
}

/**
Get clowdee's all uploaded file list.
*/
func fileList(ctx echo.Context) error {
	clowdee := ctx.Get("clowdee").(*model.Clowdee)

	fileList := &[]*FileView{}

	// find from database
	sqlResult := database.Conn().
		Table("files").
		Select("name, sum(size) as size, min(uploaded_at) as uploaded_at").
		Where("google_id = ?", clowdee.GoogleID).
		Group("name").
		Scan(fileList)

	// sql error occurred
	if sqlResult.Error != nil && !sqlResult.RecordNotFound() {
		logger.File().Errorf("Error finding the clowder's file list in database, %s", sqlResult.Error.Error())
		return ctx.NoContent(http.StatusInternalServerError)
	}

	return ctx.JSON(http.StatusOK, fileList)
}

/**
File download requested by client(clowdee).
*/
func download(ctx echo.Context) error {
	clowdee := ctx.Get("clowdee").(*model.Clowdee)

	// bind download list
	downloadList := &[]*FileToDownload{}
	if err := ctx.Bind(downloadList); err != nil {
		logger.File().Infof("Error binding client's download list, %s", err)
		return err
	}

	return nil
}
