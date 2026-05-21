package server

import (
	"bufio"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

// replica tracks one replica connection together with the last offset it has
// acknowledged via REPLCONF ACK. ackedOffset is protected by Replicas.mu.
type replica struct {
	conn        net.Conn
	ackedOffset int64
}

// Replicas is a thread-safe registry of replica connections that the master
// streams propagated write commands to. It also tracks the master replication
// offset and reads back REPLCONF ACK replies so WAIT can report durability.
type Replicas struct {
	mu        sync.Mutex
	cond      *sync.Cond
	list      []*replica
	masterOff int64
}

func newReplicas() *Replicas {
	r := &Replicas{}
	r.cond = sync.NewCond(&r.mu)
	return r
}

// Add registers a replica connection and starts reading its ACK responses.
func (r *Replicas) Add(conn net.Conn) {
	rep := &replica{conn: conn}
	r.mu.Lock()
	r.list = append(r.list, rep)
	r.mu.Unlock()
	go r.readAcks(rep)
}

// Count returns the number of currently-registered replicas.
func (r *Replicas) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.list)
}

// Broadcast writes b to every replica and advances the master replication
// offset by len(b). Replicas whose write fails are dropped.
func (r *Replicas) Broadcast(b []byte) {
	r.mu.Lock()
	snapshot := make([]*replica, len(r.list))
	copy(snapshot, r.list)
	r.masterOff += int64(len(b))
	r.mu.Unlock()

	for _, rep := range snapshot {
		if _, err := rep.conn.Write(b); err != nil {
			r.dropReplica(rep)
		}
	}
}

// Wait sends REPLCONF GETACK * to every replica (when there are pending writes)
// and waits up to timeout for at least numReplicas replicas to acknowledge the
// master's pre-GETACK offset. Returns the number of replicas that have caught
// up by the time it returns.
func (r *Replicas) Wait(numReplicas int, timeout time.Duration) int {
	r.mu.Lock()
	target := r.masterOff
	if target == 0 {
		n := len(r.list)
		r.mu.Unlock()
		return n
	}
	snapshot := make([]*replica, len(r.list))
	copy(snapshot, r.list)
	getack := []byte(resp.Array([]string{"REPLCONF", "GETACK", "*"}))
	// The GETACK we are about to send shifts each replica's processed-byte
	// counter once they handle it. Account for those bytes now so the *next*
	// WAIT uses a baseline that matches what replicas will report.
	r.masterOff += int64(len(getack))
	r.mu.Unlock()

	for _, rep := range snapshot {
		// Best-effort: a failed write is cleaned up by readAcks when the
		// connection eventually breaks.
		if _, err := rep.conn.Write(getack); err != nil {
			r.dropReplica(rep)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	timedOut := false
	timer := time.AfterFunc(timeout, func() {
		r.mu.Lock()
		timedOut = true
		r.cond.Broadcast()
		r.mu.Unlock()
	})
	defer timer.Stop()

	for {
		count := 0
		for _, rep := range r.list {
			if rep.ackedOffset >= target {
				count++
			}
		}
		if count >= numReplicas || timedOut {
			return count
		}
		r.cond.Wait()
	}
}

// readAcks runs in a goroutine, parsing REPLCONF ACK <offset> replies from the
// replica and updating its acked offset. Exits when the connection breaks.
func (r *Replicas) readAcks(rep *replica) {
	br := bufio.NewReader(rep.conn)
	for {
		parts, err := readArrayParts(br)
		if err != nil {
			r.dropReplica(rep)
			return
		}
		if len(parts) >= 3 && strings.EqualFold(parts[0], "REPLCONF") && strings.EqualFold(parts[1], "ACK") {
			off, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				log.Printf("replicas: bad ACK offset %q: %v", parts[2], err)
				continue
			}
			r.mu.Lock()
			if off > rep.ackedOffset {
				rep.ackedOffset = off
			}
			r.cond.Broadcast()
			r.mu.Unlock()
		}
	}
}

func (r *Replicas) dropReplica(rep *replica) {
	r.mu.Lock()
	for i, x := range r.list {
		if x == rep {
			r.list = append(r.list[:i], r.list[i+1:]...)
			break
		}
	}
	r.cond.Broadcast()
	r.mu.Unlock()
	rep.conn.Close()
}
