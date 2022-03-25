package scheduling

import "sync"

type queue interface {
	Enqueue(r *scheduledRequest) bool
	Dequeue() *scheduledRequest
	Len() int
}

// FIFOQueue defines a circular queue
type FIFOQueue struct {
	sync.Mutex
	data     []*scheduledRequest
	capacity int
	head     int
	tail     int
	size     int
}

// NewFIFOQueue creates a queue
func NewFIFOQueue(n int) *FIFOQueue {
	if n == 0 {
		return nil
	}
	return &FIFOQueue{
		data:     make([]*scheduledRequest, n),
		capacity: n,
		head:     0,
		tail:     0,
		size:     0,
	}
}

// IsEmpty returns true if queue is empty
func (q *FIFOQueue) IsEmpty() bool {
	return q.size == 0
}

// IsFull returns true if queue is full
func (q *FIFOQueue) IsFull() bool {
	return q.size == q.capacity
}

// Enqueue pushes an element to the back
func (q *FIFOQueue) Enqueue(v *scheduledRequest) bool {
	q.Lock()
	defer q.Unlock()

	if q.IsFull() {
		return false
	}

	q.data[q.tail] = v
	q.tail = (q.tail + 1) % q.capacity
	q.size = q.size + 1
	return true
}

// Dequeue fetches a element from queue
func (q *FIFOQueue) Dequeue() *scheduledRequest {
	q.Lock()
	defer q.Unlock()

	if q.IsEmpty() {
		return nil
	}
	v := q.data[q.head]
	q.head = (q.head + 1) % q.capacity
	q.size = q.size - 1
	return v
}

// Returns the current length of the queue
func (q *FIFOQueue) Len() int {
	return q.size
}
