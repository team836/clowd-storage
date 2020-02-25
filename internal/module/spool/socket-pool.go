package spool

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
	// mutex for all nodes' status
	NodesStatusLock sync.Mutex

	// wait group for checking all ping&pong are done
	pingWaitGroup sync.WaitGroup

	// registered nodes
	Nodes map[*ActiveNode]bool

	// register requests from the node
	Register chan *ActiveNode

	// unregister requests from the node
	Unregister chan *ActiveNode
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
		Nodes:      make(map[*ActiveNode]bool),
		Register:   make(chan *ActiveNode),
		Unregister: make(chan *ActiveNode),
	}

	// Run the pool operations concurrently
	go pool.run()

	return pool
}

func (pool *SocketPool) TotalCapacity() uint64 {
	var cap uint64 = 0
	for node := range pool.Nodes {
		cap += node.Status.Capacity
	}

	return cap
}

/**
Find active node in the pool by machine id.
*/
func (pool *SocketPool) FindActiveNode(machineID string) *ActiveNode {
	for node := range pool.Nodes {
		if node.Model.MachineID == machineID {
			return node
		}
	}

	return nil
}

/**
Send ping concurrently to nodes whose current status is old
and wait for all pong response.

This function change nodes' status. So you SHOULD use this function with
the `NodesStatusLock` which is mutex for all nodes' status.
*/
func (pool *SocketPool) CheckAllNodes() {
	now := time.Now()
	for node := range pool.Nodes {
		// check whether if the node's current status is old
		if now.After(node.Status.lastCheckedAt.Add(pingCoolTime)) {
			pool.pingWaitGroup.Add(1)
			node.Ping <- true // try ping

			select {
			// ping request is buffered
			// because this node's websocket is currently busy(processing save or load)
			case <-node.Ping:
				node.Status.isOld = true
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
Select the nodes to save the files and sort them by node selection algorithm.
Return type is ring, which is circular list, because select the nodes until
all shards are scheduled.

The first return value is the safe nodes that have latest(reliable) status.
The second return value is the unsafe nodes that have old(unreliable) status.

This function read nodes' status at specific time. So you SHOULD use this function with
the `NodesStatusLock` which is mutex for all nodes' status.
*/
func (pool *SocketPool) SelectNodes() (*ring.Ring, *ring.Ring) {
	safeNodes := make(map[*ActiveNode]bool)
	unsafeNodes := make(map[*ActiveNode]bool)

	// separate node list by whether status is old or not
	for node := range pool.Nodes {
		if node.Status.isOld {
			unsafeNodes[node] = true
		} else {
			safeNodes[node] = true
		}
	}

	// TODO: implement node selection algorithm

	return mapToRing(safeNodes), mapToRing(unsafeNodes)
}

/**
Run the pool operations using non-blocking channels.

register: register the node to pool
unregister: unregister the node from pool
*/
func (pool *SocketPool) run() {
	for {
		select {
		case node := <-pool.Register:
			pool.Nodes[node] = true

			// flush deleted shard list
			go func() {
				node.Flush <- true
			}()
		case node := <-pool.Unregister:
			_ = node.conn.Close()
			delete(pool.Nodes, node)
		}
	}
}

/**
Convert map to ring.
*/
func mapToRing(m map[*ActiveNode]bool) *ring.Ring {
	r := ring.New(len(m))
	for key := range m {
		r.Value = key
		r = r.Next()
	}

	return r
}
