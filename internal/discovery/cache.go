package discovery

import (
	"encoding/json"
	"os"
	"time"

	"github.com/oddurs/connex/internal/paths"
)

// CacheTTL bounds how long a cached peer list is reused before a fresh browse.
const CacheTTL = 5 * time.Second

type cacheFile struct {
	At    time.Time `json:"at"`
	Peers []Peer    `json:"peers"`
}

// CachedBrowse returns cached peers when the cache is younger than CacheTTL,
// otherwise performs a fresh Browse and refreshes the cache. Setting
// CONNEX_NO_CACHE forces a fresh browse every time. Only addresses are cached;
// callers must always probe for live capacity.
func CachedBrowse(timeout time.Duration) []Peer {
	if os.Getenv("CONNEX_NO_CACHE") != "" {
		return Fresh(timeout)
	}
	if peers, ok := readCache(); ok {
		return peers
	}
	return Fresh(timeout)
}

// Fresh always browses and updates the cache.
func Fresh(timeout time.Duration) []Peer {
	peers := Browse(timeout)
	writeCache(peers)
	return peers
}

func readCache() ([]Peer, bool) {
	path, err := paths.PeersCache()
	if err != nil {
		return nil, false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var c cacheFile
	if json.Unmarshal(b, &c) != nil {
		return nil, false
	}
	if time.Since(c.At) > CacheTTL {
		return nil, false
	}
	return c.Peers, true
}

func writeCache(peers []Peer) {
	path, err := paths.PeersCache()
	if err != nil {
		return
	}
	b, err := json.Marshal(cacheFile{At: time.Now(), Peers: peers})
	if err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0o600)
}
