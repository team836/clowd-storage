package node

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

	saveWait = 30 * time.Second
)

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

type FileOnNode struct {
	Name string `json:"name"`
	Data []byte `json:"data"`
}

type Node struct {
	// corresponding node model
	Model *model.Node

	// node status
	Status *Status

	// send check ping to the node and receive the node's information
	// It SHOULD be buffered channel for non-blocking at the socket pool
	Ping chan bool

	// save file on the node
	SaveFile chan []*FileOnNode

	// websocket connection
	conn *websocket.Conn
}

func NewFileOnNode(name string, data []byte) *FileOnNode {
	fon := &FileOnNode{
		Name: name,
		Data: data,
	}

	return fon
}

func NewNode(conn *websocket.Conn, model *model.Node) *Node {
	c := &Node{
		Model: model,
		Status: &Status{
			lastCheckedAt: time.Now().Add(-24 * time.Hour),
			isOld:         true,
		},
		Ping:     make(chan bool, 1), // buffered channel for trying ping
		SaveFile: make(chan []*FileOnNode),
		conn:     conn,
	}

	return c
}

/**
Run the websocket operations using non-blocking channels.
*/
func (node *Node) run() {
	defer func() {
		pool.unregister <- node
	}()

	for {
		select {
		case <-node.Ping:
			node.conn.SetReadLimit(maxPongSize)

			// send the check ping
			_ = node.conn.SetWriteDeadline(time.Now().Add(pingWait))
			if err := node.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				logger.File().Infof("Error sending ping to node, %s", err)
				pool.pingWaitGroup.Done()
				return
			}

			// receive the check pong
			_ = node.conn.SetReadDeadline(time.Now().Add(pongWait))
			if err := node.conn.ReadJSON(node.Status); err != nil {
				logger.File().Infof("Error receiving pong data from node, %s", err)
				pool.pingWaitGroup.Done()
				return
			}

			// TODO: need to update RTT

			node.Status.lastCheckedAt = time.Now() // update last ping time
			node.Status.isOld = false
			pool.pingWaitGroup.Done()
		case files := <-node.SaveFile:
			_ = node.conn.SetWriteDeadline(time.Now().Add(saveWait))
			// byte array data are send as base64 encoded format
			if err := node.conn.WriteJSON(files); err != nil {
				logger.File().Errorf("Error saving file to node, %s", err)
				return
			}
		}
	}
}
