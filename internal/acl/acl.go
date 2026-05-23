package acl

import (
	"crypto/sha256"
	"fmt"
	"sync"
)

// DefaultUser holds the ACL state for the default user, shared across all connections.
type DefaultUser struct {
	mu        sync.RWMutex
	nopass    bool
	passwords []string
}

func NewDefaultUser() *DefaultUser {
	return &DefaultUser{nopass: true}
}

func (u *DefaultUser) NoPass() bool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.nopass
}

func (u *DefaultUser) Flags() []string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	if u.nopass {
		return []string{"nopass"}
	}
	return nil
}

func (u *DefaultUser) Passwords() []string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	out := make([]string, len(u.passwords))
	copy(out, u.passwords)
	return out
}

// AddPassword hashes the plaintext password with SHA-256, clears nopass, and appends the hash.
func (u *DefaultUser) AddPassword(plaintext string) {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(plaintext)))
	u.mu.Lock()
	defer u.mu.Unlock()
	u.nopass = false
	u.passwords = append(u.passwords, hash)
}

// Authenticate returns true if the given plaintext password matches any stored hash,
// or if nopass is set (any password accepted).
func (u *DefaultUser) Authenticate(plaintext string) bool {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(plaintext)))
	u.mu.RLock()
	defer u.mu.RUnlock()
	if u.nopass {
		return true
	}
	for _, h := range u.passwords {
		if h == hash {
			return true
		}
	}
	return false
}
