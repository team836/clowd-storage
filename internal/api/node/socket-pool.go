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
		pingFlag:   make(chan bool),
	}

	go pool.run()

	return pool
}

/**
Run the pool operations concurrently using non-blocking channels.

register: register the clowder to pool
unregister: unregister the clowder from pool
ping: broadcast to all registered clowders for sending check ping
*/
func (pool *SocketPool) run() {
	for {
		select {
		case clowder := <-pool.register:
			pool.clowders[clowder] = true
		case clowder := <-pool.unregister:
			if _, ok := pool.clowders[clowder]; ok {
				delete(pool.clowders, clowder)
				close(clowder.ping)
			}
		case <-pool.pingFlag:
			for clowder := range pool.clowders {
				select {
				case clowder.ping <- true:
				default:
					close(clowder.ping)
					delete(pool.clowders, clowder)
				}
			}
		}
	}
}
