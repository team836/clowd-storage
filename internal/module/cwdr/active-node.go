package cwdr

import (
	"time"

	"github.com/team836/clowd-storage/internal/model"

	"github.com/team836/clowd-storage/pkg/logger"

	"github.com/gorilla/websocket"
)

const (
	pingWait = 500 * time.Millisecond

	pongWait = 1 * time.Second

	maxPongSize = 512

	msgWait = 500 * time.Millisecond

	saveWait = 30 * time.Second

	loadWait = 30 * time.Second

	maxLoadSize = 1048576
)

type shardToDown struct {
	Name string `json:"name"`
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
	Load chan []*model.ShardToLoad

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
		Ping: make(chan bool, 1), // buffered channel for trying ping
		Save: make(chan []*model.ShardToSave),
		Load: make(chan []*model.ShardToLoad),
		conn: conn,
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
			if err := node.conn.WriteJSON(shards); err != nil {
				logger.File().Errorf("Error saving file to node, %s", err)
				return
			}
		case shards := <-node.Load:
			node.conn.SetReadLimit(maxLoadSize)

			// make download list
			shardsToDown := &[]*shardToDown{}
			for _, shard := range shards {
				*shardsToDown = append(
					*shardsToDown,
					&shardToDown{Name: shard.Model.Name},
				)
			}

			// send the download list
			_ = node.conn.SetWriteDeadline(time.Now().Add(msgWait))
			if err := node.conn.WriteJSON(*shardsToDown); err != nil {
				logger.File().Infof("Error sending download list to node, %s", err)
				return
			}

			receivedShards := &[]*model.ShardToSave{}

			// receive the shards data
			_ = node.conn.SetReadDeadline(time.Now().Add(loadWait))
			if err := node.conn.ReadJSON(*receivedShards); err != nil {
				logger.File().Infof("Error downloading data from node, %s", err)
				return
			}

			// if count of shards is different
			if len(shards) != len(*receivedShards) {
				logger.File().Infof("Count of shards is different, maybe malformed data")
				return
			}

			// copy shard data
			for idx, receivedShard := range *receivedShards {
				// check if whether received data name is same
				if shards[idx].Model.Name != receivedShard.Name {
					logger.File().Infof("Shard name is different, maybe malformed data")
					return
				}

				shards[idx].Data = receivedShard.Data
			}
		}
	}
}
