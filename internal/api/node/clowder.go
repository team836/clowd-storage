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
)

type Status struct {
	// TODO: measure rtt when ping and pong
	// round trip time (ms)
	RTT uint `json:"rtt"`

	// network bandwidth (Mbps)
	Bandwidth uint `json:"bandwidth"`

	// available capacity of the clowder (KB)
	Capacity uint64 `json:"capacity"`
}

type Clowder struct {
	// websocket connection
	conn *websocket.Conn

	// clowder status
	status *Status
}

func NewClowder(conn *websocket.Conn) *Clowder {
	c := &Clowder{conn: conn, status: &Status{}}
	return c
}

/**
Send ping to the clowder and receive the clowder's information.

This function is run by goroutine.
*/
func (clowder *Clowder) pingPong() {
	defer pool.pingWaitGroup.Done()

	clowder.conn.SetReadLimit(maxPongSize)

	// send the check ping
	_ = clowder.conn.SetWriteDeadline(time.Now().Add(pingWait))
	if err := clowder.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
		logger.File().Infof("Error sending ping to clowder, %s", err)
		clowder.disconnect()
		return
	}

	// receive the check pong
	_ = clowder.conn.SetReadDeadline(time.Now().Add(pongWait))
	if err := clowder.conn.ReadJSON(clowder.status); err != nil {
		logger.File().Infof("Error receiving pong data from clowder, %s", err)
		clowder.disconnect()
		return
	}

	// TODO: need to update RTT
}

/**
Disconnect clowder from socket pool.
*/
func (clowder *Clowder) disconnect() {
	_ = clowder.conn.Close()
	pool.unregister <- clowder
}
