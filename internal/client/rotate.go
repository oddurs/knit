package client

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/oddurs/knit/internal/keys"
)

// Rotate replaces this machine's cluster key with a fresh one and prints it,
// naming the machines that were reachable under the old key and must now
// `knit join` the new one. This machine's own agent picks the new key up on
// its next connection; nothing needs restarting.
func Rotate() int {
	old, err := loadKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, "knit:", err)
		return 1
	}
	var names []string
	peers, _ := probePeers(old, true)
	for _, c := range peers {
		names = append(names, c.Name)
	}
	key, err := keys.Rotate()
	if err != nil {
		fmt.Fprintln(os.Stderr, "knit:", err)
		return 1
	}
	fmt.Println(hex.EncodeToString(key))
	switch len(names) {
	case 0:
		fmt.Fprintln(os.Stderr, "knit: new key installed; no other machines were reachable under the old one")
	default:
		fmt.Fprintf(os.Stderr, "knit: new key installed; run `knit join <key>` on: %s\n", strings.Join(names, ", "))
	}
	return 0
}
