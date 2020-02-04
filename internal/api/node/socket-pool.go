package node

import "sync"

type SocketPool struct {
	// registered clowders
	clowders map[*Clowder]bool

	// register requests from the clowder
	register chan *Clowder

	// unregister requests from the clowder
	unregister chan *Clowder

	// flag to send check ping to all clowders
	pingFlag chan bool
}

var (
	pool *SocketPool // singleton instance
	once sync.Once   // for thread safe singleton
)

/**
Return the singleton socket pool instance.
*/
func Pool() *SocketPool {
	once.Do(func() {
		pool = newSocketPool()
	})

	return pool
}

func newSocketPool() *SocketPool {
	pool := &SocketPool{
		clowders:   make(map[*Clowder]bool),
		register:   make(chan *Clowder),
		unregister: make(chan *Clowder),
	}

	return pool
}
