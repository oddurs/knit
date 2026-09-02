// Package proto defines the knit wire protocol: the KNIT1 handshake,
// the JSON control messages, and the length-prefixed stdio framing.
//
// One TCP connection per operation. The server opens with
//
//	KNIT1 <32-hex nonce>\n
//
// the client replies with one Request line, and the server replies with one
// Envelope line. For op "run" the client then streams raw stdin while the
// server streams typed frames back. See docs/03-protocol.md.
package proto

// Version is the protocol version the client advertises in Request.V. v3 is
// the TLS generation: the connection is TLS 1.3 and both proofs are bound to
// it. A v2 agent cannot be reached by a v3 client, and vice versa; each side
// says so plainly instead of hanging.
const Version = 3

// Magic is the greeting token prefixing the server nonce line.
const Magic = "KNIT1"

// Ops.
const (
	OpInfo = "info"
	OpRun  = "run"
)

// Frame types. The server->client stream uses 1-3; the client->server stream
// (KNIT2, protocol V2) uses 10-12 so the two directions never collide in a log.
const (
	// server -> client
	FrameStdout byte = 1
	FrameStderr byte = 2
	FrameExit   byte = 3

	// client -> server (V2)
	FrameStdin    byte = 10
	FrameStdinEOF byte = 11
	FrameSignal   byte = 12 // payload: 1 byte signal number (2=SIGINT, 15=SIGTERM)
	// 13 (winsize) is reserved.

	// directory transfer for --dir/--sync (KN-EXEC-030)
	FrameInTar  byte = 20 // client -> server: --dir tree, tar chunk
	FrameInEnd  byte = 21 // client -> server: end of --dir tree
	FrameOutTar byte = 30 // server -> client: --sync mirror-back, tar chunk
	FrameOutEnd byte = 31 // server -> client: end of --sync mirror-back
)

// MaxFrame is the largest payload a single frame may carry.
const MaxFrame = 1 << 20 // 1 MiB

// Stable machine-readable error codes carried in Envelope.Code.
const (
	CodeUnauthorized = "unauthorized"
	CodeVersion      = "version"
	CodeEmptyCmd     = "empty_cmd"
	CodeSpawn        = "spawn"
	CodeInternal     = "internal"
)

// Request is the single JSON line a client sends after the nonce.
type Request struct {
	V    int      `json:"v"`
	HMAC string   `json:"hmac"`
	Op   string   `json:"op"`
	Cmd  []string `json:"cmd,omitempty"`
	// Dir sends the client's working directory to the target and runs there
	// (KN-EXEC-030); Sync mirrors changed files back on completion.
	Dir  bool `json:"dir,omitempty"`
	Sync bool `json:"sync,omitempty"`
	// Hosts, when set, is the rank-ordered address list of a `knit each`
	// launch and Rank this target's position in it (KN-AI-030). The agent
	// exposes both to the command as environment (see RankEnv).
	Hosts []string `json:"hosts,omitempty"`
	Rank  int      `json:"rank,omitempty"`
}

// Envelope is the single JSON line the server sends in reply. For op "info"
// it carries the capacity fields; for op "run" it is just {ok:true} (or an
// error) and streaming follows.
type Envelope struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Code  string `json:"code,omitempty"`
	// Proof is the server's HMAC over the nonce and this connection's channel
	// binding; the client checks it so an impostor agent is refused.
	Proof string `json:"proof,omitempty"`

	// Capacity, populated for op "info".
	Name      string  `json:"name,omitempty"`
	OS        string  `json:"os,omitempty"`
	Arch      string  `json:"arch,omitempty"`
	CPUs      int     `json:"cpus,omitempty"`
	MemGB     float64 `json:"mem_gb,omitempty"`
	MemFreeGB float64 `json:"mem_free_gb,omitempty"`
	Load1     float64 `json:"load1,omitempty"`
	Accel     string  `json:"accel,omitempty"`
	GPU       string  `json:"gpu,omitempty"`

	// Self marks the local machine in gauge output; never sent on the wire.
	Self bool `json:"self,omitempty"`
	// Link is how the client reaches this peer ("thunderbolt ~40G", "wifi",
	// "lan", "net"); derived from the peer address, never sent by the agent.
	Link string `json:"link,omitempty"`
}

// Score is load per core: lower means more spare capacity. A machine that
// reports zero cores sorts last.
func (e Envelope) Score() float64 {
	if e.CPUs == 0 {
		return 1e9
	}
	return e.Load1 / float64(e.CPUs)
}
