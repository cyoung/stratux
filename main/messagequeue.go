/*
	Copyright (c) 2021 Adrian Batzill
	Distributable under the terms of The "BSD New" License
	that can be found in the LICENSE file, herein included
	as part of this header.

	messagequeue.go: Prioritizing queue for slow network connections
*/

package main

import (
	"sort"
	"sync"
	"time"
)

type QueueEntry struct {
	priority   int32
	outdatedAt time.Time
	data       interface{}
}


type MessageQueue struct {
	maxSize       int
	entries       []QueueEntry
	DataAvailable chan bool
	Closed        bool
	mutex         sync.Mutex
}

func NewMessageQueue(maxSize int) *MessageQueue {
	return &MessageQueue {
		maxSize: maxSize,
		entries: make([]QueueEntry, 0),
		DataAvailable: make(chan bool, 1),
	}
}

func (queue *MessageQueue) Put(prio int32, maxAge time.Duration, data interface{}) {
	if queue.Closed {
		return
	}
	queue.mutex.Lock()
	defer queue.mutex.Unlock()

	timeout := stratuxClock.Time.Add(maxAge)
	entry := QueueEntry { prio, timeout, data }

	if queue.entries == nil || len(queue.entries) == 0 {
		queue.entries = make([]QueueEntry, 1)
		queue.entries[0] = QueueEntry { prio, timeout, data }
	} else {
		index := queue.findInsertPosition(prio)
		
		if index == len(queue.entries) {
			queue.entries = append(queue.entries, entry)
		} else {
			queue.entries = append(queue.entries[:index+1], queue.entries[index:]...)
			queue.entries[index] = entry
		}
	}

	if len(queue.entries) > queue.maxSize {
		queue.prune()
	}
	if len(queue.entries) != 0 {
		queue.notifyData()
	}
}

func (queue *MessageQueue) PeekFirst() (interface{}, int32) {
	return queue.getFirst(false)
}


func (queue *MessageQueue) PopFirst() (interface{}, int32) {
	return queue.getFirst(true)
}

func (queue *MessageQueue) getFirst(remove bool) (interface{}, int32) {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()

	index := queue.getFirstUsableIndex()
	if index < 0 {
		return nil, 0 // nothing in queue
	}

	// found one. Strip the queue and return it
	entry := queue.entries[index]
	if remove  {
		queue.entries = queue.entries[index+1:]
	} else {
		queue.entries = queue.entries[index:]
	}
	return entry.data, entry.priority
}

// Returns the first entry that's not outdated
func (queue *MessageQueue) getFirstUsableIndex() int {
	for i, data := range queue.entries {
		if data.outdatedAt.Before(stratuxClock.Time) {
			// entry already timed out..
			continue
		}
		// found one
		return i
	}

	// Nothing current in queue
	if len(queue.entries) > 0 {
		queue.entries = make([]QueueEntry, 0)
	}

	return -1
}

func (queue *MessageQueue) GetQueueDump(pruneFirst bool) []interface{} {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()

	if pruneFirst {
		queue.prune()
	}
	data := make([]interface{}, len(queue.entries))
	for i, d := range queue.entries {
		data[i] = d.data
	}
	return data
}

func (queue *MessageQueue) prune() {
	newEntries := make([]QueueEntry, 0)
	//npruned := 0
	for _, entry := range queue.entries {
		if entry.outdatedAt.After(stratuxClock.Time) {
			newEntries = append(newEntries, entry)
		} else {
			//npruned++
			
		}
		if len(newEntries) == int(queue.maxSize) {
			break
		}
	}
	queue.entries = newEntries
	//if npruned > 0 {
	//	fmt.Printf("Pruned %d entries\n", npruned)
	//}
}

func (queue *MessageQueue) findInsertPosition(priority int32) int {
	index := sort.Search(len(queue.entries), func(i int) bool {
		// > instead of >= so we get to the first entry that is larger - in order to keep insertion order
		// for equal-priority messages
		return queue.entries[i].priority > priority
	})

	return index
}

func (queue *MessageQueue) notifyData() {
	select {
	case queue.DataAvailable <- true:
	default:
	}
}

func (queue *MessageQueue) Close() {
	if queue.Closed {
		return
	}
	queue.Closed = true
	queue.notifyData()
}