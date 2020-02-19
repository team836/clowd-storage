package node

import (
	"container/ring"
	"sync"
	"time"
)

const (
	pingCoolTime = 3 * time.Second
)

var (
	pool *SocketPool // singleton instance
	once sync.Once   // for thread safe singleton
)

type SocketPool struct {
	// mutex for all clowders' status
	ClowdersStatusLock sync.Mutex

	// wait group for checking all ping&pong are done
	pingWaitGroup sync.WaitGroup

	// registered clowders
	clowders map[*Clowder]bool

	// register requests from the clowder
	register chan *Clowder

	// unregister requests from the clowder
	unregister chan *Clowder
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

/**
Create new socket pool.
*/
func newSocketPool() *SocketPool {
	pool := &SocketPool{
		clowders:   make(map[*Clowder]bool),
		register:   make(chan *Clowder),
		unregister: make(chan *Clowder),
	}

	// run the pool operations concurrently
	go pool.run()

	return pool
}

func (pool *SocketPool) TotalCapacity() uint64 {
	var cap uint64 = 0
	for clowder := range pool.clowders {
		cap += clowder.Status.Capacity
	}

	return cap
}

/**
Send ping concurrently to clowders whose current status is old
and wait for all pong response.

This function change clowders' status. So you SHOULD use this function with
the `ClowdersStatusLock` which is mutex for all clowders' status.
*/
func (pool *SocketPool) CheckAllClowders() {
	now := time.Now()
	for clowder := range pool.clowders {
		// check whether if the clowder's current status is old
		if now.After(clowder.Status.lastCheckedAt.Add(pingCoolTime)) {
			pool.pingWaitGroup.Add(1)
			clowder.Ping <- true // try ping

			select {
			// ping request is buffered
			// because this clowder's websocket is currently busy(processing save or load)
			case <-clowder.Ping:
				clowder.Status.isOld = true
				pool.pingWaitGroup.Done()
			// concurrently process ping request
			default:
			}
		}
	}

	// wait for all ping&pong to finish
	pool.pingWaitGroup.Wait()
}

/**
Select the clowders to save the files and sort them by node selection algorithm.
Return type is ring, which is circular list, because select the clowders until
all shards are scheduled.

This function read clowders' status at specific time. So you SHOULD use this function with
the `ClowdersStatusLock` which is mutex for all clowders' status.
*/
func (pool *SocketPool) SelectClowders() *ring.Ring {
	// TODO: implement node selection algorithm
	r := ring.New(len(pool.clowders))

	for clowder := range pool.clowders {
		r.Value = clowder
		r = r.Next()
	}

	return r
}

/**
Run the pool operations using non-blocking channels.

register: register the clowder to pool
unregister: unregister the clowder from pool
*/
func (pool *SocketPool) run() {
	for {
		select {
		case clowder := <-pool.register:
			pool.clowders[clowder] = true
		case clowder := <-pool.unregister:
			delete(pool.clowders, clowder)
		}
	}
}
