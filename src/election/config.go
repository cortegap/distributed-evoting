package election

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

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
	// saved     []*Persister
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
	cfg.connected = make([]bool, cfg.nCounters)
	cfg.counterEndnames = make([][]string, cfg.nCounters)
	cfg.voterEndnames = make([][]string, cfg.nVoters)

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

	return cfg
}

func (cfg *config) startCounter(i int) {

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

	vc := MakeVoteCounter(ends, i, cfg.nCounters, cfg.nVoters, cfg.nCounters)

	cfg.mu.Lock()
	cfg.counters[i] = vc
	cfg.mu.Unlock()

	svc := labrpc.MakeService(vc)
	srv := labrpc.MakeServer()
	srv.AddService(svc)
	cfg.net.AddServer(i, srv)
}

func (cfg *config) startVoter(i, vote int) {

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

	vt := MakeVoter(ends, cfg.nCounters, vote)

	cfg.mu.Lock()
	cfg.voters[i] = vt
	cfg.mu.Unlock()
}

// attach server i to the net.
func (cfg *config) connectCounter(i int) {
	// fmt.Printf("connect(%d)\n", i)

	cfg.connected[i] = true

	// outgoing ClientEnds
	for j := 0; j < cfg.nCounters; j++ {
		if cfg.connected[j] {
			endname := cfg.counterEndnames[i][j]
			cfg.net.Enable(endname, true)
		}
	}

	// incoming counter ClientEnds
	for j := 0; j < cfg.nCounters; j++ {
		if cfg.connected[j] {
			endname := cfg.counterEndnames[j][i]
			cfg.net.Enable(endname, true)
		}
	}

	// incoming voter ClientEnds
	for j := 0; j < cfg.nVoters; j++ {
		endname := cfg.voterEndnames[j][i]
		cfg.net.Enable(endname, true)
	}
}

func (cfg *config) voteResult() int {
	for iters := 0; iters < 10; iters++ {
		time.Sleep(1000 * time.Millisecond)

		for i := 0; i < cfg.nCounters; i++ {
			done, vote := cfg.counters[i].Done()
			if done {
				return vote
			}
		}
	}
	return -1
}

func (cfg *config) setunreliable(unrel bool) {
	cfg.net.Reliable(!unrel)
}
