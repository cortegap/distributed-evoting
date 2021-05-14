package election

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"math/rand"
	"runtime"
	"sync"
	"testing"

	crand "crypto/rand"

	"6.824/labrpc"
)

type config struct {
	mu               sync.Mutex
	t                *testing.T
	net              *labrpc.Network
	nCounters        int
	nVoters          int
	counters         []*VoteCounter
	voters           []*Voter
	counterConnected []bool // whether each server is on the net
	voterConnected   []bool
	counterEndnames  [][]string // the port file names each sends to
	voterEndnames    [][]string
	saved            []*Persister
}

type VoterPersister struct {
	mu         sync.Mutex
	voteShares []byte
}

func (vp *VoterPersister) readPersistState() []byte {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	return vp.voteShares
}

func (vp *VoterPersister) writePersistState(data []byte) {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	vp.voteShares = data
}

func makeSeed() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := crand.Int(crand.Reader, max)
	x := bigx.Int64()
	return x
}

func randstring(n int) string {
	b := make([]byte, 2*n)
	crand.Read(b)
	s := base64.URLEncoding.EncodeToString(b)
	return s[0:n]
}

var ncpu_once sync.Once

func makeConfig(t *testing.T, nCounters, nVoters int, votes []int, unreliable bool) *config {
	ncpu_once.Do(func() {
		if runtime.NumCPU() < 2 {
			fmt.Printf("warning: only one CPU, which may conceal locking bugs\n")
		}
		rand.Seed(makeSeed())
	})
	runtime.GOMAXPROCS(4)

	cfg := &config{}
	cfg.t = t
	cfg.net = labrpc.MakeNetwork()
	cfg.nCounters = nCounters
	cfg.nVoters = nVoters
	cfg.counters = make([]*VoteCounter, cfg.nCounters)
	cfg.voters = make([]*Voter, cfg.nVoters)
	cfg.counterConnected = make([]bool, cfg.nCounters)
	cfg.voterConnected = make([]bool, cfg.nVoters)
	cfg.counterEndnames = make([][]string, cfg.nCounters)
	cfg.voterEndnames = make([][]string, cfg.nVoters)
	cfg.saved = make([]*Persister, cfg.nVoters)

	cfg.setunreliable(unreliable)

	for i := 0; i < cfg.nCounters; i++ {
		cfg.startCounter(i)
	}

	for i := 0; i < cfg.nVoters; i++ {
		cfg.startVoter(i, votes[i])
	}

	for i := 0; i < cfg.nCounters; i++ {
		cfg.connectCounter(i)
	}

	for i := 0; i < cfg.nVoters; i++ {
		cfg.connectVoter(i)
	}

	return cfg
}

func (cfg *config) startCounter(i int) {
	cfg.crashCounter(i)

	// a fresh set of outgoing ClientEnd names.
	// so that old crashed instance's ClientEnds can't send.
	cfg.counterEndnames[i] = make([]string, cfg.nCounters)
	for j := 0; j < cfg.nCounters; j++ {
		cfg.counterEndnames[i][j] = randstring(20)
	}

	// a fresh set of ClientEnds.
	ends := make([]*labrpc.ClientEnd, cfg.nCounters)
	for j := 0; j < cfg.nCounters; j++ {
		ends[j] = cfg.net.MakeEnd(cfg.counterEndnames[i][j])
		cfg.net.Connect(cfg.counterEndnames[i][j], j)
	}

	vc := MakeVoteCounter(ends, i, cfg.nVoters, cfg.nCounters)

	cfg.mu.Lock()
	cfg.counters[i] = vc
	cfg.mu.Unlock()

	svc := labrpc.MakeService(vc)
	srv := labrpc.MakeServer()
	srv.AddService(svc)
	cfg.net.AddServer(i, srv)
}

func (cfg *config) crashCounter(i int) {
	cfg.disconnectCounter(i)
	cfg.net.DeleteServer(i) // disable client connections to the server.

	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	vc := cfg.counters[i]
	if vc != nil {
		cfg.mu.Unlock()
		vc.Kill()
		cfg.mu.Lock()
		cfg.counters[i] = nil
	}
}

func (cfg *config) startVoter(i, vote int) {
	cfg.crashVoter(i)

	// a fresh set of outgoing ClientEnd names.
	// so that old crashed instance's ClientEnds can't send.
	cfg.voterEndnames[i] = make([]string, cfg.nCounters)
	for j := 0; j < cfg.nCounters; j++ {
		cfg.voterEndnames[i][j] = randstring(20)
	}

	// a fresh set of ClientEnds.
	ends := make([]*labrpc.ClientEnd, cfg.nCounters)
	for j := 0; j < cfg.nCounters; j++ {
		ends[j] = cfg.net.MakeEnd(cfg.voterEndnames[i][j])
		cfg.net.Connect(cfg.voterEndnames[i][j], j)
	}

	vt := MakeVoter(ends, vote, cfg.nCounters, *cfg.saved[i])

	cfg.mu.Lock()
	cfg.voters[i] = vt
	cfg.mu.Unlock()
}

func (cfg *config) startVoting() {
	for i := 0; i < cfg.nVoters; i++ {
		cfg.voters[i].Vote()
	}
}

func (cfg *config) crashVoter(i int) {
	cfg.disconnectVoter(i)

	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	vt := cfg.voters[i]
	if vt != nil {
		cfg.mu.Unlock()
		vt.Kill()
		cfg.mu.Lock()
		cfg.voters[i] = nil
	}
}

// attach server i to the net.
func (cfg *config) connectCounter(i int) {
	// fmt.Printf("connect(%d)\n", i)

	cfg.counterConnected[i] = true

	// outgoing ClientEnds
	for j := 0; j < cfg.nCounters; j++ {
		if cfg.counterConnected[j] {
			endname := cfg.counterEndnames[i][j]
			cfg.net.Enable(endname, true)
		}
	}

	// incoming counter ClientEnds
	for j := 0; j < cfg.nCounters; j++ {
		if cfg.counterConnected[j] {
			endname := cfg.counterEndnames[j][i]
			cfg.net.Enable(endname, true)
		}
	}

	// incoming voter ClientEnds
	for j := 0; j < cfg.nVoters; j++ {
		if cfg.voterConnected[j] {
			endname := cfg.voterEndnames[j][i]
			cfg.net.Enable(endname, true)
		}
	}
}

// attach server i to the net.
func (cfg *config) disconnectCounter(i int) {
	// fmt.Printf("connect(%d)\n", i)

	cfg.counterConnected[i] = false

	// outgoing ClientEnds
	for j := 0; j < cfg.nCounters; j++ {
		if cfg.counterEndnames[i] != nil {
			endname := cfg.counterEndnames[i][j]
			cfg.net.Enable(endname, false)
		}
	}

	// incoming counter ClientEnds
	for j := 0; j < cfg.nCounters; j++ {
		if cfg.counterEndnames[j] != nil {
			endname := cfg.counterEndnames[j][i]
			cfg.net.Enable(endname, false)
		}
	}

	// incoming voter ClientEnds
	for j := 0; j < cfg.nVoters; j++ {
		if cfg.voterEndnames[j] != nil {
			endname := cfg.voterEndnames[j][i]
			cfg.net.Enable(endname, false)
		}
	}
}

func (cfg *config) connectVoter(i int) {
	// fmt.Printf("connect(%d)\n", i)

	cfg.voterConnected[i] = true

	// outgoing voter ClientEnds
	for j := 0; j < cfg.nCounters; j++ {
		if cfg.counterConnected[j] {
			endname := cfg.voterEndnames[i][j]
			cfg.net.Enable(endname, true)
		}
	}
}

func (cfg *config) disconnectVoter(i int) {
	// fmt.Printf("connect(%d)\n", i)

	cfg.voterConnected[i] = false

	// outgoing voter ClientEnds
	for j := 0; j < cfg.nCounters; j++ {
		if cfg.voterEndnames[i] != nil {
			endname := cfg.voterEndnames[i][j]
			cfg.net.Enable(endname, false)
		}
	}
}

func (cfg *config) setunreliable(unrel bool) {
	cfg.net.Reliable(!unrel)
}

func (cfg *config) cleanup() {
	for i := range cfg.counters {
		if cfg.counters[i] != nil {
			cfg.counters[i].Kill()
		}
	}
	for i := range cfg.voters {
		if cfg.voters[i] != nil {
			cfg.voters[i].Kill()
		}
	}

	cfg.net.Cleanup()
}
