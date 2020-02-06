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

	// ping requests
	ping chan bool

	// clowder status
	status *Status
}

/**
Wait for the check ping flag and then send ping to the clowder
and receive the clowder's information.

A goroutine running this function is started for each connection.
*/
func (clowder *Clowder) onPingPong() {
	defer func() {
		pool.unregister <- clowder
		_ = clowder.conn.Close()
	}()

	clowder.conn.SetReadLimit(maxPongSize)

	for {
		_, ok := <-clowder.ping // wait for the ping requested
		_ = clowder.conn.SetWriteDeadline(time.Now().Add(pingWait))

		// The pool closed the channel
		if !ok {
			_ = clowder.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		// send the check ping
		err := clowder.conn.WriteMessage(websocket.PingMessage, []byte{})
		if err != nil {
			logger.File().Info("Error sending ping to clowder, %s", err)
			return
		}

		// receive the check pong
		_ = clowder.conn.SetReadDeadline(time.Now().Add(pongWait))
		_, msg, err := clowder.conn.ReadMessage()
		if err != nil {
			logger.File().Info("Error receiving pong from clowder, %s", err)
			return
		}

		// TODO: Need to handle the received pong message
		logger.Console().Printf("pong message: %s", msg)
	}
}
