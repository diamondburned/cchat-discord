package nonce

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	cryptorand "crypto/rand"
	mathrand "math/rand"
)

func init() {
	mathrand.Seed(time.Now().UnixNano())
}

var nonceCounter uint64

// generateNonce generates a unique nonce ID.
func generateNonce() string {
	return fmt.Sprintf(
		"%s-%s-%s",
		strconv.FormatInt(time.Now().Unix(), 36),
		randomBits(),
		strconv.FormatUint(atomic.AddUint64(&nonceCounter, 1), 36),
	)
}

// randomBits returns a string 6 bytes long with random characters that are safe
// to print. It falls back to math/rand's pseudorandom number generator if it
// cannot read from the system entropy pool.
func randomBits() string {
	randBits := make([]byte, 2)

	_, err := cryptorand.Read(randBits)
	if err != nil {
		binary.LittleEndian.PutUint32(randBits, mathrand.Uint32())
	}

	return base64.RawStdEncoding.EncodeToString(randBits)
}

// Map is a nonce state that keeps track of known nonces and generates a
// Discord-compatible nonce string.
type Map sync.Map

// Generate generates a new internal nonce, add a bind from the new nonce to the
// original nonce, then return the new nonce. If the given original nonce is
// empty, then an empty string is returned.
func (nmap *Map) Generate(original string) string {
	// Ignore empty nonces.
	if original == "" {
		return ""
	}

	newNonce := generateNonce()
	(*sync.Map)(nmap).Store(newNonce, original)
	return newNonce
}

// Load grabs the nonce and permanently deleting it if the given nonce is found.
func (nmap *Map) Load(newNonce string) string {
	v, ok := (*sync.Map)(nmap).LoadAndDelete(newNonce)
	if ok {
		return v.(string)
	}
	return ""
}

// Set is a unique set of nonces.
type Set sync.Map

var nonceSentinel = struct{}{}

func (nset *Set) Store(nonce string) {
	(*sync.Map)(nset).Store(nonce, nonceSentinel)
}

func (nset *Set) Has(nonce string) bool {
	_, ok := (*sync.Map)(nset).Load(nonce)
	return ok
}

func (nset *Set) HasAndDelete(nonce string) bool {
	_, ok := (*sync.Map)(nset).LoadAndDelete(nonce)
	return ok
}
