package server

import (
	"bufio"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

// pipePair returns one half wired into Replicas (master side) and the other
// for the test goroutine to act as the replica.
func pipePair(t *testing.T) (master, replica net.Conn) {
	t.Helper()
	a, b := net.Pipe()
	t.Cleanup(func() {
		a.Close()
		b.Close()
	})
	return a, b
}

// simulateReplica drains everything sent by the master, and on every
// REPLCONF GETACK replies with REPLCONF ACK <ack>. Stops when the conn closes.
func simulateReplica(t *testing.T, c net.Conn, ack int64) {
	t.Helper()
	go func() {
		br := bufio.NewReader(c)
		for {
			parts, err := readArrayParts(br)
			if err != nil {
				return
			}
			if len(parts) >= 2 && strings.EqualFold(parts[0], "REPLCONF") && strings.EqualFold(parts[1], "GETACK") {
				_, _ = c.Write([]byte(resp.Array([]string{
					"REPLCONF", "ACK", strconv.FormatInt(ack, 10),
				})))
			}
		}
	}()
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", msg)
}

func TestReplicasBroadcastWritesToAll(t *testing.T) {
	r := newReplicas()
	mA, rA := pipePair(t)
	mB, rB := pipePair(t)
	r.Add(mA)
	r.Add(mB)

	var wg sync.WaitGroup
	got := make([]string, 2)
	for i, c := range []net.Conn{rA, rB} {
		i, c := i, c
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 5)
			n, _ := c.Read(buf)
			got[i] = string(buf[:n])
		}()
	}
	r.Broadcast([]byte("hello"))
	wg.Wait()

	for i, s := range got {
		if s != "hello" {
			t.Errorf("replica %d got %q, want %q", i, s, "hello")
		}
	}
}

func TestReplicasCountDropsOnReplicaClose(t *testing.T) {
	r := newReplicas()
	master, replica := pipePair(t)
	r.Add(master)
	if r.Count() != 1 {
		t.Fatalf("count=%d, want 1", r.Count())
	}
	replica.Close()
	waitFor(t, func() bool { return r.Count() == 0 }, "replica to be dropped")
}

func TestReplicasWaitNoWritesReturnsCountImmediately(t *testing.T) {
	r := newReplicas()
	m, _ := pipePair(t)
	r.Add(m)

	start := time.Now()
	n := r.Wait(5, 500*time.Millisecond)
	elapsed := time.Since(start)
	if n != 1 {
		t.Errorf("got %d, want 1", n)
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("Wait should be immediate when no writes, took %s", elapsed)
	}
}

func TestReplicasWaitReturnsWhenAllReplicasAck(t *testing.T) {
	r := newReplicas()
	mA, rA := pipePair(t)
	mB, rB := pipePair(t)
	r.Add(mA)
	r.Add(mB)

	// Replicas drain the broadcast then ack with the post-broadcast offset.
	go func() {
		buf := make([]byte, 1024)
		rA.Read(buf)
	}()
	go func() {
		buf := make([]byte, 1024)
		rB.Read(buf)
	}()
	writeCmd := []byte(resp.Array([]string{"SET", "foo", "bar"}))
	r.Broadcast(writeCmd)

	simulateReplica(t, rA, int64(len(writeCmd)))
	simulateReplica(t, rB, int64(len(writeCmd)))

	start := time.Now()
	n := r.Wait(2, time.Second)
	elapsed := time.Since(start)
	if n != 2 {
		t.Errorf("got %d, want 2", n)
	}
	if elapsed >= time.Second {
		t.Errorf("Wait should have returned before timeout, took %s", elapsed)
	}
}

func TestReplicasWaitTimesOutWithPartialAcks(t *testing.T) {
	r := newReplicas()
	mA, rA := pipePair(t)
	mB, rB := pipePair(t)
	r.Add(mA)
	r.Add(mB)

	go func() {
		buf := make([]byte, 1024)
		rA.Read(buf)
	}()
	go func() {
		buf := make([]byte, 1024)
		rB.Read(buf)
	}()
	writeCmd := []byte(resp.Array([]string{"SET", "foo", "bar"}))
	r.Broadcast(writeCmd)

	// Only one replica will respond with a valid ack; the other only drains.
	simulateReplica(t, rA, int64(len(writeCmd)))
	go func() {
		buf := make([]byte, 1024)
		for {
			if _, err := rB.Read(buf); err != nil {
				return
			}
		}
	}()

	start := time.Now()
	n := r.Wait(2, 150*time.Millisecond)
	elapsed := time.Since(start)
	if n != 1 {
		t.Errorf("got %d, want 1", n)
	}
	if elapsed < 150*time.Millisecond {
		t.Errorf("Wait should have blocked until timeout, took %s", elapsed)
	}
}

func TestReplicasWaitReturnsEarlyOnceTargetReached(t *testing.T) {
	r := newReplicas()
	mA, rA := pipePair(t)
	mB, rB := pipePair(t)
	mC, rC := pipePair(t)
	r.Add(mA)
	r.Add(mB)
	r.Add(mC)

	for _, c := range []net.Conn{rA, rB, rC} {
		c := c
		go func() {
			buf := make([]byte, 1024)
			c.Read(buf)
		}()
	}
	writeCmd := []byte(resp.Array([]string{"SET", "foo", "bar"}))
	r.Broadcast(writeCmd)

	simulateReplica(t, rA, int64(len(writeCmd)))
	simulateReplica(t, rB, int64(len(writeCmd)))
	// rC never acks.
	go func() {
		buf := make([]byte, 1024)
		for {
			if _, err := rC.Read(buf); err != nil {
				return
			}
		}
	}()

	start := time.Now()
	n := r.Wait(2, 2*time.Second)
	elapsed := time.Since(start)
	if n < 2 {
		t.Errorf("got %d, want >=2", n)
	}
	if elapsed >= 2*time.Second {
		t.Errorf("Wait should have returned once 2 acks arrived, took %s", elapsed)
	}
}
