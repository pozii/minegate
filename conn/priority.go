package conn

import (
	"container/list"
	"sync"

	"github.com/user/minegate/internal"
	"github.com/user/minegate/packet"
)

// Priority is the packet priority level.
type Priority int

const (
	Urgent Priority = iota // Keepalive, state changes
	Normal                 // Movement, game packets
	Low                    // Chunk data, map data
)

// PriorityItem is a priority queue element.
type PriorityItem struct {
	Data     packet.Packet
	Priority Priority
	Size     int
}

// PriorityQueue is a multi-level priority queue.
// High-priority packets are always sent first.
type PriorityQueue struct {
	levels []*list.List
	cap    []int
	mu     sync.Mutex
	notify chan struct{}
	closed bool
}

// NewPriorityQueue creates an n-level priority queue.
func NewPriorityQueue(levels int) *PriorityQueue {
	pq := &PriorityQueue{
		levels: make([]*list.List, levels),
		cap:    make([]int, levels),
		notify: make(chan struct{}, 1),
	}
	for i := range pq.levels {
		pq.levels[i] = list.New()
		pq.cap[i] = 256 // default capacity
	}
	return pq
}

// Push adds a prioritized item.
func (pq *PriorityQueue) Push(item *PriorityItem) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.closed {
		return internal.ErrConnectionClosed
	}

	level := int(item.Priority)
	if level >= len(pq.levels) {
		level = len(pq.levels) - 1
	}

	pq.levels[level].PushBack(item)

	select {
	case pq.notify <- struct{}{}:
	default:
	}

	return nil
}

// Pop retrieves the highest-priority item.
func (pq *PriorityQueue) Pop() (*PriorityItem, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	empty := true
	for _, l := range pq.levels {
		if l.Len() > 0 {
			empty = false
			break
		}
	}

	if pq.closed && empty {
		return nil, internal.ErrConnectionClosed
	}

	for _, l := range pq.levels {
		if l.Len() > 0 {
			elem := l.Front()
			l.Remove(elem)
			return elem.Value.(*PriorityItem), nil
		}
	}

	return nil, internal.ErrQueueEmpty
}

// PopBlock blocks until an item is available, returning the highest-priority one.
func (pq *PriorityQueue) PopBlock() (*PriorityItem, error) {
	for {
		pq.mu.Lock()
		if pq.closed && pq.Len() == 0 {
			pq.mu.Unlock()
			return nil, internal.ErrConnectionClosed
		}
		for _, l := range pq.levels {
			if l.Len() > 0 {
				elem := l.Front()
				l.Remove(elem)
				pq.mu.Unlock()
				return elem.Value.(*PriorityItem), nil
			}
		}
		pq.mu.Unlock()

		<-pq.notify
	}
}

// Len returns the total item count across all levels.
func (pq *PriorityQueue) Len() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	total := 0
	for _, l := range pq.levels {
		total += l.Len()
	}
	return total
}

// Close closes the queue.
func (pq *PriorityQueue) Close() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.closed = true
	close(pq.notify)
}

var _ = packet.Packet{}
