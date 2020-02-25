package spool

import (
	"sync"
	"time"

	"github.com/team836/clowd-storage/pkg/database"

	"github.com/team836/clowd-storage/internal/model"

	"github.com/team836/clowd-storage/pkg/logger"

	"github.com/gorilla/websocket"
)

const (
	pingWait = 500 * time.Millisecond

	pongWait = 1 * time.Second

	msgSendWait = 500 * time.Millisecond

	saveWait = 30 * time.Second

	loadWait = 30 * time.Second
)

const (
	maxPongSize = 512

	maxLoadSize = 104857600 // 100MB
)

const (
	uploadType = "save"

	downloadType = "down"

	deleteType = "delete"
)

type shardToDown struct {
	Name string `json:"name"`
}

type DataMsg struct {
	Type     string      `json:"type"`
	Contents interface{} `json:"contents"`
}

type LoadChan struct {
	Shards []*model.ShardToLoad
	WG     *sync.WaitGroup
}

type Status struct {
	// TODO: measure rtt when ping and pong
	// round trip time (ms)
	RTT uint `json:"rtt"`

	// network bandwidth (Mbps)
	Bandwidth uint `json:"bandwidth"`

	// available capacity of the node (Byte)
	Capacity uint64 `json:"capacity"`

	// last checked time for this status
	lastCheckedAt time.Time

	// whether this status is old or not
	isOld bool
}

type ActiveNode struct {
	// corresponding node model
	Model *model.Node

	// node status
	Status *Status

	// send check ping to the node and receive the node's information
	// It SHOULD be buffered channel for non-blocking at the socket pool
	Ping chan bool

	// save shards on the node
	Save chan []*model.ShardToSave

	// load shards from the node
	Load chan *LoadChan

	// delete shards on the node
	Delete chan []*model.ShardToDelete

	// flush the deleted shard list to the node
	Flush chan bool

	// websocket connection
	conn *websocket.Conn
}

func NewActiveNode(conn *websocket.Conn, nodeModel *model.Node) *ActiveNode {
	c := &ActiveNode{
		Model: nodeModel,
		Status: &Status{
			lastCheckedAt: time.Now().Add(-24 * time.Hour),
			isOld:         true,
		},
		Ping:   make(chan bool, 1), // buffered channel for trying ping
		Save:   make(chan []*model.ShardToSave),
		Load:   make(chan *LoadChan),
		Delete: make(chan []*model.ShardToDelete),
		Flush:  make(chan bool),
		conn:   conn,
	}

	return c
}

/**
Run the websocket operations using non-blocking channels.
*/
func (node *ActiveNode) Run() {
	defer func() {
		Pool().Unregister <- node
	}()

	for {
		select {
		case <-node.Ping:
			node.conn.SetReadLimit(maxPongSize)

			// send the check ping
			_ = node.conn.SetWriteDeadline(time.Now().Add(pingWait))
			if err := node.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				logger.File().Infof("Error sending ping to node, %s", err)
				Pool().pingWaitGroup.Done()
				return
			}

			// receive the check pong
			_ = node.conn.SetReadDeadline(time.Now().Add(pongWait))
			if err := node.conn.ReadJSON(node.Status); err != nil {
				logger.File().Infof("Error receiving pong data from node, %s", err)
				Pool().pingWaitGroup.Done()
				return
			}

			// TODO: need to update RTT

			node.Status.lastCheckedAt = time.Now() // update last ping time
			node.Status.isOld = false
			Pool().pingWaitGroup.Done()
		case shards := <-node.Save:
			_ = node.conn.SetWriteDeadline(time.Now().Add(saveWait))

			// byte array data are send as base64 encoded format
			if err := node.conn.WriteJSON(DataMsg{Type: uploadType, Contents: shards}); err != nil {
				logger.File().Errorf("Error saving file to node, %s", err)
				return
			}
		case loadChan := <-node.Load:
			node.conn.SetReadLimit(maxLoadSize)

			// make download list
			shardsToDown := make([]*shardToDown, 0)
			for _, shard := range loadChan.Shards {
				shardsToDown = append(
					shardsToDown,
					&shardToDown{Name: shard.Model.Name},
				)
			}

			// send the download list
			_ = node.conn.SetWriteDeadline(time.Now().Add(msgSendWait))
			if err := node.conn.WriteJSON(DataMsg{Type: downloadType, Contents: shardsToDown}); err != nil {
				logger.File().Infof("Error sending download list to node, %s", err)
				loadChan.WG.Done()
				return
			}

			receivedShards := make([]*model.ShardToSave, 0)

			// receive the shards data
			_ = node.conn.SetReadDeadline(time.Now().Add(loadWait))
			if err := node.conn.ReadJSON(&receivedShards); err != nil {
				logger.File().Infof("Error downloading data from node, %s", err)
				loadChan.WG.Done()
				return
			}

			// if count of shards is different
			if len(loadChan.Shards) != len(receivedShards) {
				logger.File().Infof("Count of shards is different, maybe malformed data")
				loadChan.WG.Done()
				return
			}

			// copy shard data
			for idx, receivedShard := range receivedShards {
				// check if whether received data name is same
				if loadChan.Shards[idx].Model.Name != receivedShard.Name {
					logger.File().Infof("Shard name is different, maybe malformed data")
					loadChan.WG.Done()
					return
				}

				loadChan.Shards[idx].Data = receivedShard.Data
			}

			loadChan.WG.Done()
		case shards := <-node.Delete:
			_ = node.conn.SetWriteDeadline(time.Now().Add(msgSendWait))

			// send the deletion list
			if err := node.conn.WriteJSON(DataMsg{Type: deleteType, Contents: shards}); err != nil {
				// record to database for later deletion because currently cannot delete
				for _, shard := range shards {
					database.Conn().
						Create(&model.DeletedShard{Name: shard.Name, MachineID: node.Model.MachineID})
				}

				logger.File().Errorf("Error sending deletion list to node, %s", err)
				return
			}
		case <-node.Flush:
			_ = node.conn.SetWriteDeadline(time.Now().Add(msgSendWait))

			flushList := make([]*model.DeletedShard, 0)

			// get flush list from the database
			database.Conn().
				Where("machine_id = ?", node.Model.MachineID).
				Find(&flushList)

			// if flush list is not empty
			if len(flushList) != 0 {
				// make deletion list
				shards := make([]*model.ShardToDelete, 0)
				for _, delShard := range flushList {
					shards = append(shards, &model.ShardToDelete{Name: delShard.Name})
				}

				// send the deletion list to the node
				if err := node.conn.WriteJSON(DataMsg{Type: deleteType, Contents: shards}); err != nil {
					logger.File().Errorf("Error flushing deletion list to node, %s", err)
					return
				}

				// at this point, deletion is success
				// so delete records of deletion list
				for _, flushedShard := range flushList {
					database.Conn().
						Delete(flushedShard)
				}
			}
		}
	}
}
