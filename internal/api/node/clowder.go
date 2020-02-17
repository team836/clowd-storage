package node

import (
	"time"

	"github.com/team836/clowd-storage/pkg/logger"

	"github.com/gorilla/websocket"
)

const (
	pingWait = 500 * time.Millisecond

	pongWait = 1 * time.Second

	maxPongSize = 512

	saveWait = 15 * time.Second
)

type Status struct {
	// TODO: measure rtt when ping and pong
	// round trip time (ms)
	RTT uint `json:"rtt"`

	// network bandwidth (Mbps)
	Bandwidth uint `json:"bandwidth"`

	// available capacity of the clowder (Byte)
	Capacity uint64 `json:"capacity"`
}

type FileOnNode struct {
	Name string `json:"name"`
	Data []byte `json:"data"`
}

type Clowder struct {
	// clowder status
	Status *Status

	// send check ping to the clowder and receive the clowder's information
	// It SHOULD be buffered channel for non-blocking at the socket pool
	Ping chan bool

	// save file on the clowder
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

func NewClowder(conn *websocket.Conn) *Clowder {
	c := &Clowder{
		Status:   &Status{},
		Ping:     make(chan bool, 1), // buffered channel
		SaveFile: make(chan []*FileOnNode),
		conn:     conn,
	}

	return c
}

/**
Run the websocket operations using non-blocking channels.
*/
func (clowder *Clowder) run() {
	defer func() {
		_ = clowder.conn.Close()
		pool.unregister <- clowder
	}()

	for {
		select {
		case <-clowder.Ping:
			clowder.conn.SetReadLimit(maxPongSize)

			// send the check ping
			_ = clowder.conn.SetWriteDeadline(time.Now().Add(pingWait))
			if err := clowder.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				logger.File().Infof("Error sending ping to clowder, %s", err)
				pool.pingWaitGroup.Done()
				return
			}

			// receive the check pong
			_ = clowder.conn.SetReadDeadline(time.Now().Add(pongWait))
			if err := clowder.conn.ReadJSON(clowder.Status); err != nil {
				logger.File().Infof("Error receiving pong data from clowder, %s", err)
				pool.pingWaitGroup.Done()
				return
			}

			// TODO: need to update RTT

			pool.pingWaitGroup.Done()
		case files := <-clowder.SaveFile:
			_ = clowder.conn.SetWriteDeadline(time.Now().Add(saveWait))
			// byte array data are send as base64 encoded format
			if err := clowder.conn.WriteJSON(files); err != nil {
				logger.File().Errorf("Error saving file to clowder, %s", err)
				return
			}
		}
	}
}
