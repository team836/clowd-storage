package node

import (
	"sync"
	"time"
)

const (
	pingCoolTime = 20 * time.Second
)

var (
	pool *SocketPool // singleton instance
	once sync.Once   // for thread safe singleton
)

type SocketPool struct {
	// mutex for all clowders' status
	ClowdersStatusLock sync.Mutex

	// registered clowders
	clowders map[*Clowder]bool

	// register requests from the clowder
	register chan *Clowder

	// unregister requests from the clowder
	unregister chan *Clowder

	// flag to send check ping to all clowders
	pingFlag chan bool

	// last ping time
	lastPingAt time.Time

	// wait group for checking all ping&pong are done
	pingWaitGroup sync.WaitGroup
}

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
		lastPingAt: time.Now().Add(-24 * time.Hour),
	}

	go pool.run()

	return pool
}

/**
Run the pool operations concurrently using non-blocking channels.

register: register the clowder to pool
unregister: unregister the clowder from pool
pingFlag: send ping concurrently to all registered clowders for check clowders' status
*/
func (pool *SocketPool) run() {
	for {
		select {
		case clowder := <-pool.register:
			pool.clowders[clowder] = true
		case clowder := <-pool.unregister:
			delete(pool.clowders, clowder)
		case <-pool.pingFlag:
			// check ping cool time
			if time.Now().After(pool.lastPingAt.Add(pingCoolTime)) {
				pool.lastPingAt = time.Now()

				for clowder := range pool.clowders {
					pool.pingWaitGroup.Add(1)
					go clowder.pingPong()
				}
			}

			pool.pingWaitGroup.Done()
		}
	}
}

/**
Encapsulated function for sending ping flag and
wait for all ping&pong to finish.
*/
func (pool *SocketPool) CheckAllClowders() {
	pool.pingWaitGroup.Add(1)
	pool.pingFlag <- true
	pool.pingWaitGroup.Wait()
}
