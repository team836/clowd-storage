package client

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4/middleware"

	"github.com/team836/clowd-storage/internal/module/operationq"
	"github.com/team836/clowd-storage/internal/module/spool"

	"github.com/team836/clowd-storage/pkg/database"

	"github.com/team836/clowd-storage/internal/model"

	"github.com/team836/clowd-storage/pkg/errcorr"

	"github.com/team836/clowd-storage/pkg/logger"

	"github.com/labstack/echo/v4"
)

const (
	uploadLimit = "100M"
)

type fileOnClient struct {
	Name  string `json:"name"`
	Order int    `json:"order"`
	Data  string `json:"data"` // base64 encoded
}

type fileView struct {
	Name       string    `json:"name"`
	Size       uint      `json:"size"`
	UploadedAt time.Time `json:"uploadedAt"`
}

type fileToDown struct {
	Name string `json:"name"`
}

type fileToDelete struct {
	fileToDown
}

func RegisterHandlers(group *echo.Group) {
	group.GET("/dir", fileListController)
	group.POST("/files", uploadController, middleware.BodyLimit(uploadLimit))
	group.GET("/files", downloadController)
	group.DELETE("/files", deleteController)
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
func uploadController(ctx echo.Context) error {
	clowdee := ctx.Get("clowdee").(*model.Clowdee)

	// bind uploaded data into array of `fileOnClient` struct
	files := make([]*fileOnClient, 0)
	if err := ctx.Bind(&files); err != nil {
		logger.File().Infof("Error binding client's uploaded file, %s", err)
		return err
	}

	// create upload queue
	uq := operationq.NewUQ()

	// encode every file data using reed solomon algorithm
	// and push to upload queue
	for _, file := range files {
		// encode the file data
		shards, size, err := errcorr.Encode(file.Data)
		if err != nil {
			logger.File().Infof("Error encoding the file, %s", err)
			return ctx.String(http.StatusNotAcceptable, "Cannot handle this file: "+file.Name)
		}

		encFile := &model.EncFile{
			Model: &model.File{
				GoogleID: clowdee.GoogleID,
				Name:     file.Name,
				Position: int16(file.Order),
				Size:     size,
			},
			Data: shards,
		}

		// if the file is already exists
		if !database.Conn().NewRecord(encFile.Model) {
			logger.File().Infof("The file is already exists")
			return ctx.String(http.StatusNotAcceptable, `"`+encFile.Model.Name+`" is already exists`)
		}

		// push to upload queue
		uq.Push(encFile)
	}

	// this area almost change all nodes' status
	// so, protect it using mutex for all node's status
	spool.Pool().NodesStatusLock.Lock()
	defer func() {
		// when panic is occurred, unlock the nodes status lock
		if r := recover(); r != nil {
			spool.Pool().NodesStatusLock.Unlock()
		}
	}()

	// check all nodes' status by ping&pong
	spool.Pool().CheckAllNodes()

	// node selection
	safeRing, unsafeRing := spool.Pool().SelectNodes()
	if safeRing.Len()+unsafeRing.Len() == 0 {
		spool.Pool().NodesStatusLock.Unlock()
		logger.File().Errorf("Available nodes are not exist.")
		return ctx.String(http.StatusNotAcceptable, "Cannot save the files because currently there are no available nodes")
	}

	// schedule saving for every shards to the nodes
	// and get results
	quotas, err := uq.Schedule(safeRing, unsafeRing)
	if err != nil {
		spool.Pool().NodesStatusLock.Unlock()
		logger.File().Errorf("Error scheduling upload, %s", err)

		if err == operationq.ErrLackOfStorage {
			return ctx.String(http.StatusNotAcceptable, err.Error())
		}

		return ctx.NoContent(http.StatusInternalServerError)
	}

	// save each quota using goroutine
	for nodeToSave, shards := range quotas {
		go func(a *spool.ActiveNode, s []*model.ShardToSave) {
			a.Save <- s
		}(nodeToSave, shards)
	}

	// end of mutex area for nodes status lock
	spool.Pool().NodesStatusLock.Unlock()

	return ctx.NoContent(http.StatusCreated)
}

/**
Get clowdee's all uploaded file list.
*/
func fileListController(ctx echo.Context) error {
	clowdee := ctx.Get("clowdee").(*model.Clowdee)

	fileList := make([]*fileView, 0)

	// find from database
	sqlResult := database.Conn().
		Table("files").
		Select("name, sum(size) as size, min(uploaded_at) as uploaded_at").
		Where("google_id = ?", clowdee.GoogleID).
		Group("name").
		Scan(&fileList)

	// sql error occurred
	if sqlResult.Error != nil && !sqlResult.RecordNotFound() {
		logger.File().Errorf("Error finding the node's file list in database, %s", sqlResult.Error.Error())
		return ctx.NoContent(http.StatusInternalServerError)
	}

	return ctx.JSON(http.StatusOK, &fileList)
}

/**
File download requested by client(clowdee).
*/
func downloadController(ctx echo.Context) error {
	clowdee := ctx.Get("clowdee").(*model.Clowdee)

	// bind download list from header
	downloadList := make([]*fileToDown, 0)
	if err := json.Unmarshal([]byte(ctx.Request().Header.Get("files")), &downloadList); err != nil {
		logger.File().Infof("Error binding client's download list, %s", err)
		return ctx.String(http.StatusNotAcceptable, "Invalid request format")
	}

	dq := operationq.NewDQ()

	// read download list and add them to download queue
	for _, file := range downloadList {
		if err := dq.Push(clowdee.GoogleID, file.Name); err != nil {
			if err == operationq.ErrFileNotExist {
				return ctx.String(http.StatusNotFound, err.Error()+": "+file.Name)
			}

			return ctx.NoContent(http.StatusInternalServerError)
		}
	}

	// schedule every shards for download to the each active nodes
	// and get quotas for each nodes
	quotas := dq.Schedule()

	// concurrently download each quota using goroutine
	var downloadWG sync.WaitGroup
	for machineID, shards := range quotas {
		// if the machine is active
		if activeNode := spool.Pool().FindActiveNode(machineID); activeNode != nil {
			downloadWG.Add(1)

			// start new worker for download
			go func(a *spool.ActiveNode, s []*model.ShardToLoad, wg *sync.WaitGroup) {
				a.Load <- &spool.LoadChan{Shards: s, WG: wg}
			}(activeNode, shards, &downloadWG)
		}
	}

	// wait for all download workers are done
	downloadWG.Wait()

	response := make([]*fileOnClient, 0)
	reconstructedShards := make([]*model.ShardToLoad, 0)

	// make response data
	for _, file := range dq.Files {
		var shards [][]byte
		var missedShards []*model.ShardToLoad

		// merge all shard data
		// if the shard is invalid, make to nil for reconstruction
		for _, loadedShard := range file.Shards {
			// if shard is missing or corrupted, make to nil data
			if len(loadedShard.Data) == 0 ||
				errcorr.IsCorruptedChecksum(loadedShard.Data, loadedShard.Model.Checksum) {
				loadedShard.Data = nil
			}

			// if shard data is missed, add to missed list
			if loadedShard.Data == nil {
				missedShards = append(missedShards, loadedShard)
			}

			shards = append(shards, loadedShard.Data)
		}

		// reconstruct the original file from the shards
		fileData, reconstructedShardData, err := errcorr.Decode(shards, int(file.Model.Size))
		if err != nil {
			logger.File().Infof("Error decoding the shards, %s", err)
			return ctx.String(http.StatusInternalServerError, "file download error")
		}

		// insert reconstructed data to the missed list
		for idx, missedShard := range missedShards {
			missedShard.Data = reconstructedShardData[idx]
		}

		// merge to all missed list
		reconstructedShards = append(reconstructedShards, missedShards...)

		response = append(
			response,
			&fileOnClient{
				Name:  file.Model.Name,
				Order: int(file.Model.Position),
				Data:  fileData,
			},
		)
	}

	go restoreShards(reconstructedShards)

	return ctx.JSON(http.StatusOK, &response)
}

/**
Restore(re-upload) the reconstruct shards to the another nodes.
*/
func restoreShards(reconstructedShards []*model.ShardToLoad) {
	// there are not exists shards to restore
	if len(reconstructedShards) == 0 {
		return
	}

	rq := operationq.NewRQ()
	rq.Push(reconstructedShards...)

	spool.Pool().NodesStatusLock.Lock()
	defer spool.Pool().NodesStatusLock.Unlock()

	spool.Pool().CheckAllNodes()

	// node selection
	safeRing, unsafeRing := spool.Pool().SelectNodes()
	if safeRing.Len()+unsafeRing.Len() == 0 {
		logger.File().Errorf("Available nodes are not exist.")
		return
	}

	// schedule restoring for every shards to the nodes
	// and get results
	quotas, err := rq.Schedule(safeRing, unsafeRing)
	if err != nil {
		logger.File().Errorf("Error scheduling restoring, %s", err)
		return
	}

	// save each quota using goroutine
	for nodeToSave, restoreShards := range quotas {
		go func(a *spool.ActiveNode, s []*model.ShardToSave) {
			a.Save <- s
		}(nodeToSave, restoreShards)
	}
}

/**
Controller for file deletion request.
*/
func deleteController(ctx echo.Context) error {
	clowdee := ctx.Get("clowdee").(*model.Clowdee)

	// bind deletion list from the request body
	files := make([]*fileToDelete, 0)
	if err := ctx.Bind(&files); err != nil {
		logger.File().Infof("Error binding client's delete list, %s", err)
		return err
	}

	// make file name list
	nameList := make([]string, 0)
	for _, file := range files {
		nameList = append(nameList, file.Name)
	}

	delQ := operationq.NewDelQ()

	// add deletion list to delete queue
	if err := delQ.Push(clowdee.GoogleID, nameList...); err != nil {
		if err != operationq.ErrFileNotExist {
			return ctx.NoContent(http.StatusInternalServerError)
		}
	}

	// schedule every shards for deletion to the each active nodes
	// and get quotas for each nodes
	quotas := delQ.Schedule()

	// delete quotas concurrently
	for machineID, shards := range quotas {
		// if the machine is active
		if activeNode := spool.Pool().FindActiveNode(machineID); activeNode != nil {
			// delete shards on the node concurrently
			go func(a *spool.ActiveNode, s []*model.ShardToDelete) {
				a.Delete <- s
			}(activeNode, shards)
		} else { // if the machine is not active (cannot delete currently)
			// record to database for later deletion
			for _, shard := range shards {
				database.Conn().
					Create(&model.DeletedShard{Name: shard.Name, MachineID: machineID})
			}
		}
	}

	return ctx.NoContent(http.StatusNoContent)
}
