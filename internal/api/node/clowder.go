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

	// send check ping to the clowder and receive the clowder's information
	// It SHOULD be buffered channel for non-blocking at the socket pool
	Ping chan bool
}

func NewClowder(conn *websocket.Conn) *Clowder {
	c := &Clowder{conn: conn, status: &Status{}, Ping: make(chan bool, 1)}
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
			if err := clowder.conn.ReadJSON(clowder.status); err != nil {
				logger.File().Infof("Error receiving pong data from clowder, %s", err)
				pool.pingWaitGroup.Done()
				return
			}

			// TODO: need to update RTT

			pool.pingWaitGroup.Done()
		}
	}
}
