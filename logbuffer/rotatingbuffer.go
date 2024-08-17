package logbuffer

import (
	"slices"
	"strings"
	"sync"
)

// RotatingLogBuffer splits incoming logs based on the write
// number of lines are limited to a specified number, then cuts off the oldest entry
//
// Example:
//
//	limit = 3
//	buf[0] = "hello"
//	buf[1] = "world"
//	buf[2] = "!"
//
// When another is written:
//
//	limit = 3
//	buf[0] = "Lorum"
//	buf[1] = "hello"
//	buf[2] = "world"
type RotatingBuffer struct {
	mu sync.Mutex

	data [][]byte
}

// NewRotatingBuffer initalized with limit
func NewRotatingBuffer(limit int) *RotatingBuffer {
	return &RotatingBuffer{
		data: make([][]byte, limit),
	}
}

// shiftSlice taking every element and moving it one spot closer to index zero
// shiftSlice([]int{1, 2, 3, 4, 5}) = []int{2, 3, 4, 5, 5}
func shiftSlice[T any](t []T) {
	for i := 1; i < len(t); i++ {
		t[i-1] = t[i]
	}
}

// Write b to our current buffer. Will never return anything other than len(b) and nil.
func (rb *RotatingBuffer) Write(b []byte) (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	shiftSlice(rb.data)

	rb.data[len(rb.data)-1] = slices.Clone(b)

	return len(b), nil
}

// String returns all the currently stored byte slices concatentated together into a string.
func (rb *RotatingBuffer) String() string {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	b := strings.Builder{}

	b.Grow(len(rb.data))

	for i := range rb.data {
		b.Write(rb.data[i])
	}

	return b.String()
}
