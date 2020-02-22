package client

import (
	"net/http"
	"time"

	"github.com/team836/clowd-storage/internal/module/cwde"
	"github.com/team836/clowd-storage/internal/module/cwdr"

	"github.com/team836/clowd-storage/pkg/database"

	"github.com/team836/clowd-storage/internal/model"

	"github.com/team836/clowd-storage/pkg/errcorr"

	"github.com/team836/clowd-storage/pkg/logger"

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
- Check all nodes' status by ping and pong.
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
	uq := cwde.NewUQ()

	// encode every file data using reed solomon algorithm
	for _, file := range *files {
		shards, err := errcorr.Encode(file.Data)
		if err != nil {
			logger.File().Infof("Error encoding the file, %s", err)
			return ctx.String(http.StatusNotAcceptable, "Cannot handle this file: "+file.Name)
		}

		encFile := &cwde.EncFile{
			Model: &model.File{
				GoogleID: clowdee.GoogleID,
				Name:     file.Name,
				Position: int16(file.Order),
				Size:     uint(len(file.Data)),
			},
			Data: shards,
		}
		uq.Push(encFile)
	}

	// this area almost change all nodes' status
	// so, protect it using mutex for all node's status
	cwdr.Pool().NodesStatusLock.Lock()
	defer func() {
		// when panic is occurred, unlock the nodes status lock
		if r := recover(); r != nil {
			cwdr.Pool().NodesStatusLock.Unlock()
		}
	}()

	// check all nodes' status by ping&pong
	cwdr.Pool().CheckAllNodes()

	// node selection
	safeRing, unsafeRing := cwdr.Pool().SelectNodes()
	if safeRing.Len()+unsafeRing.Len() == 0 {
		cwdr.Pool().NodesStatusLock.Unlock()
		logger.File().Errorf("Available nodes are not exist.")
		return ctx.String(http.StatusNotAcceptable, "Cannot save the files because currently there are no available nodes")
	}

	// schedule saving for every shards to the nodes
	// and get results
	quotas, err := uq.Schedule(safeRing, unsafeRing)
	if err != nil {
		cwdr.Pool().NodesStatusLock.Unlock()
		logger.File().Errorf("Error scheduling upload, %s", err)

		if err == cwde.ErrLackOfStorage {
			return ctx.String(http.StatusNotAcceptable, err.Error())
		}

		return ctx.NoContent(http.StatusInternalServerError)
	}

	// save each quota using goroutine
	for nodeToSave, shards := range quotas {
		go func(n *cwdr.ActiveNode, s []*cwdr.ShardOnNode) {
			n.Save <- s
		}(nodeToSave, shards)
	}

	// end of mutex area for nodes status lock
	cwdr.Pool().NodesStatusLock.Unlock()

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
		logger.File().Errorf("Error finding the node's file list in database, %s", sqlResult.Error.Error())
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

	dq := cwde.NewDQ()

	// read download list and add them to download queue
	for _, file := range *downloadList {
		fileModel := &model.File{
			GoogleID: clowdee.GoogleID,
			Name:     file.Name,
		}

		if err := dq.Push(fileModel); err != nil {
			if err == cwde.ErrFileNotExist {
				return ctx.String(http.StatusNotFound, err.Error()+": "+file.Name)
			}
		}
	}

	//// schedule every shards for download to the each nodes
	//quotas := dq.Schedule()
	//
	//// download each quota using goroutine
	//for machineID, shards := range quotas {
	//
	//}

	return nil
}
