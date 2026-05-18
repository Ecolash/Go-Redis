# BLPOP Implementation

## What BLPOP Does

`BLPOP key [key ...] timeout` is the blocking variant of `LPOP`. It atomically pops the first available element from the leftmost non-empty list among the given keys. If no list has an element, it blocks until one appears or the timeout expires.

- **timeout = 0** → block indefinitely  
- **timeout > 0** → block at most that many seconds (fractional seconds supported)  
- **Response on success** → a two-element array `[key, value]`  
- **Response on timeout** → null array (`*-1\r\n`)

---

## Architecture Overview

The implementation is split across two layers:

| Layer | File | Responsibility |
|-------|------|----------------|
| Handler | `internal/handler/handler.go` | Parse args, manage timeout, format response |
| Store | `internal/store/store.go` | Atomic pop, waiter registration, delivery on push |

---

## Store Layer

### Data Structures

```go
// BLPOPResult is what a waiter receives when an element becomes available.
type BLPOPResult struct {
    Key string
    Val string
}

// blpopWaiter tracks one pending BLPOP client watching one or more keys.
type blpopWaiter struct {
    keys []string
    ch   chan BLPOPResult // buffered(1): sender never blocks
}

type Store struct {
    mu      sync.RWMutex
    data    map[string]entry
    wmu     sync.Mutex               // guards waiters; always acquired after mu
    waiters map[string][]*blpopWaiter // key → ordered list of waiters
}
```

The channel is **buffered with size 1** so the push path can fire-and-forget without blocking even if the waiter has already timed out and stopped reading.

### `BLPopWait(keys []string) (<-chan BLPOPResult, func())`

This is the entry point called by the handler. It does two things atomically under `s.mu`:

**1. Immediate pop (fast path)**

Iterates the keys in order. If any key has a non-empty list, it pops the head element, sends the result directly into the channel, and returns. The cancel function is a no-op because no waiter was registered.

```
for each key in priority order:
    if list[key] is non-empty:
        pop head → send into w.ch
        return (ch, noop-cancel)
```

**2. Register waiter (slow path)**

If no key had data, the waiter is appended to `s.waiters[key]` for **every** key. This is how one BLPOP call can watch multiple keys simultaneously.

The returned `cancel` function removes the waiter from all key queues (under `wmu`). The handler calls this via `defer cancel()` so cleanup always happens — on timeout, on success, and on error.

### `deliverToWaitersLocked(key string)`

Called at the end of every `LPush` / `RPush` while still holding `s.mu` write lock. It loops, matching the head waiter to the head list element:

```
while waiters[key] is non-empty AND list[key] is non-empty:
    waiter  = waiters[key][0]
    val     = list[key][0]
    pop val from list
    remove waiter from ALL keys it was watching
    send BLPOPResult{key, val} into waiter.ch
```

Removing the waiter from all its keys prevents it from receiving a second delivery if it was watching multiple keys.

### Lock Ordering

To prevent deadlocks, the lock order is always:

```
s.mu (write) → s.wmu
```

`deliverToWaitersLocked` acquires `wmu` while already holding `mu`. `BLPopWait`'s slow path mirrors this: it acquires `mu`, then `wmu`, then releases both.

---

## Handler Layer

```go
func (h *Handler) handleBLPop(parts []string) string {
    // parts: [BLPOP, key1, ..., keyN, timeout]
    keys := parts[1 : len(parts)-1]
    timeoutSecs, _ := strconv.ParseFloat(parts[len(parts)-1], 64)

    channel, cancel := h.store.BLPopWait(keys)
    defer cancel()

    if timeoutSecs == 0 {
        result := <-channel          // block forever
        return resp.Array([]string{result.Key, result.Val})
    }

    timer := time.NewTimer(...)
    defer timer.Stop()

    select {
    case result := <-channel:
        return resp.Array([]string{result.Key, result.Val})
    case <-timer.C:
        return nullArray             // "$-1\r\n" — timed out
    }
}
```

The `select` on a buffered channel works correctly for the fast path too: if data was pre-loaded into the channel inside `BLPopWait`, the `case result := <-channel` arm fires immediately without waiting for the timer.

---

## End-to-End Flow

### Scenario 1: Data already present

```
Client: BLPOP mylist 5
  → BLPopWait: acquires mu, finds mylist non-empty
  → pops head, sends into buffered channel, releases mu
  → handler: select fires immediately on channel case
  → returns [mylist, value]
```

### Scenario 2: Blocking then push

```
Client A: BLPOP mylist 30
  → BLPopWait: mylist empty → registers waiter, returns channel

Client B: LPUSH mylist hello
  → LPush acquires mu, prepends element
  → calls deliverToWaitersLocked("mylist")
  → finds Client A's waiter, pops element, sends into waiter.ch
  → removes waiter from all key registrations

Client A: select receives from channel
  → returns [mylist, hello]
```

### Scenario 3: Timeout

```
Client: BLPOP mylist 2
  → BLPopWait: mylist empty → registers waiter
  → handler: select blocks on both channel and timer

  (2 seconds pass, no push)

  → timer.C fires
  → defer cancel() removes waiter from waiters["mylist"]
  → returns null array
```

---

## Key Design Decisions

**Buffered channel (size 1)**: The push path sends without blocking. A timed-out client's channel just has an orphaned value that gets GC'd — no goroutine leak.

**Priority ordering**: Keys are checked and served left-to-right, matching Redis semantics where the leftmost key with data wins.

**Waiter registered for all keys**: A single `blpopWaiter` appears in `s.waiters` for every key it watches. The first push to any of those keys delivers the result and removes the waiter from all queues, so it can't be served twice.

**`defer cancel()`**: Unconditional cleanup in the handler means waiter entries are never leaked, regardless of how the handler exits.
