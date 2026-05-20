package server

import (
	"io"
	"sync"
)

// Replicas is a thread-safe registry of replica connections that the master
// streams propagated write commands to.
type Replicas struct {
	mu    sync.Mutex
	conns []io.Writer
}

func newReplicas() *Replicas {
	return &Replicas{}
}

// Add registers a replica's write end. The same w is used for all subsequent
// broadcasts.
func (r *Replicas) Add(w io.Writer) {
	r.mu.Lock()
	r.conns = append(r.conns, w)
	r.mu.Unlock()
}

// Count returns the number of currently-registered replicas.
func (r *Replicas) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.conns)
}

func (r *Replicas) Broadcast(b []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	alive := r.conns[:0]
	for _, w := range r.conns {
		if _, err := w.Write(b); err == nil {
			alive = append(alive, w)
		}
	}
	r.conns = alive
}
