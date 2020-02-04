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

type Clowder struct {
	// websocket connection
	conn *websocket.Conn

	// flag to send check ping to this clowder
	pingFlag chan bool
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

	// connection setting
	_ = clowder.conn.SetWriteDeadline(time.Now().Add(pingWait))
	_ = clowder.conn.SetReadDeadline(time.Now().Add(pongWait))
	clowder.conn.SetReadLimit(maxPongSize)

	for {
		_, ok := <-clowder.pingFlag // wait for the ping flag

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
		_, msg, err := clowder.conn.ReadMessage()
		if err != nil {
			logger.File().Info("Error receiving pong from clowder, %s", err)
			return
		}

		// TODO: Need to handle the received pong message
		logger.Console().Printf("pong message: %s", msg)
	}
}
