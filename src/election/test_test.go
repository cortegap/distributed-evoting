package election

import (
	"fmt"
	"testing"
	"time"
)

func (cfg *config) voteResult() int {
	for iters := 0; iters < 10; iters++ {
		time.Sleep(1000 * time.Millisecond)

		for i := 0; i < cfg.nCounters; i++ {
			if cfg.counters[i] != nil {
				done, vote := cfg.counters[i].Done()
				if done {
					return vote
				}
			}
		}
	}
	return -1
}

func TestInitialElection0(t *testing.T) {
	fmt.Println("Starting simple test - 0 wins")
	cfg := makeConfig(t, 3, 5, 3, []int{0, 0, 0, 1, 1}, false)
	cfg.startVoting()

	voteResult := cfg.voteResult()
	if voteResult != 0 {
		cfg.t.Fatalf("expecting result of 0, but got %v", voteResult)
	} else {
		fmt.Println("ok")
	}

	cfg.cleanup()
}

func TestInitialElection1(t *testing.T) {
	fmt.Println("Starting simple test - 1 wins")
	cfg := makeConfig(t, 3, 5, 3, []int{0, 0, 1, 1, 1}, false)
	cfg.startVoting()

	voteResult := cfg.voteResult()
	if voteResult != 1 {
		cfg.t.Fatalf("expecting result of 1, but got %v", voteResult)
	} else {
		fmt.Println("ok")
	}

	cfg.cleanup()
}

func TestUnreliableElection0(t *testing.T) {
	fmt.Println("Starting unreliable election test - 0 wins")
	cfg := makeConfig(t, 3, 5, 3, []int{0, 0, 0, 1, 1}, true)
	cfg.startVoting()

	voteResult := cfg.voteResult()
	if voteResult != 0 {
		cfg.t.Fatalf("expecting result of 0, but got %v", voteResult)
	} else {
		fmt.Println("ok")
	}

	cfg.cleanup()
}

func TestUnreliableElection1(t *testing.T) {
	fmt.Println("Starting unreliable election test - 1 wins")
	cfg := makeConfig(t, 3, 5, 3, []int{0, 0, 1, 1, 1}, true)
	cfg.startVoting()

	voteResult := cfg.voteResult()
	if voteResult != 1 {
		cfg.t.Fatalf("expecting result of 1, but got %v", voteResult)
	} else {
		fmt.Println("ok")
	}

	cfg.cleanup()
}

// Test voter crash without recovery
func TestInfiniteElection(t *testing.T) {
	fmt.Println("Starting infinite election test")
	cfg := makeConfig(t, 3, 5, 3, []int{0, 0, 1, 1, 1}, true)
	cfg.crashVoter(2)

	cfg.startVoting()
	time.Sleep(time.Duration(5000) * time.Millisecond)

	voteResult := cfg.voteResult()
	if voteResult != -1 {
		cfg.t.Fatalf("expecting no result, but got %v", voteResult)
	} else {
		fmt.Println("ok")
	}

	cfg.cleanup()
}

// Test voter crash with recovery
func TestPersistedVoteElection(t *testing.T) {
	fmt.Println("Starting persisted vote election test - 1 wins")
	cfg := makeConfig(t, 3, 5, 3, []int{0, 0, 1, 1, 1}, true)
	cfg.crashVoter(2)

	cfg.startVoting()

	cfg.startVoter(2)
	cfg.connectVoter(2)
	cfg.vote(2)

	voteResult := cfg.voteResult()
	if voteResult != 1 {
		cfg.t.Fatalf("expecting result of 1, but got %v", voteResult)
	} else {
		fmt.Println("ok")
	}

	cfg.cleanup()
}

// Test crasher with servers
func TestServerCrash(t *testing.T) {
	fmt.Println("Starting server crash result test - 1 wins")
	cfg := makeConfig(t, 5, 7, 3, []int{0, 0, 0, 1, 1, 1, 1}, false)

	randomServer := int(nrand(5))
	cfg.crashCounter(randomServer)
	cfg.crashCounter((randomServer + 1) % 5)
	cfg.crashCounter((randomServer + 2) % 5)

	cfg.startVoting()

	voteNoResult := cfg.voteResult()
	if voteNoResult != -1 {
		cfg.t.Fatalf("expecting no result, but got %v", voteNoResult)
	}

	cfg.startCounter((randomServer + 1) % 5)
	cfg.connectCounter((randomServer + 1) % 5)

	voteResult := cfg.voteResult()
	if voteResult != 1 {
		cfg.t.Fatalf("expecting 1, but got %v", voteResult)
	} else {
		fmt.Println("ok")
	}

	cfg.cleanup()
}

// Test crasher with servers and unreliable network
func TestServerCrashUnreliable(t *testing.T) {
	fmt.Println("Starting server crash unreliable result test - 0 wins")
	cfg := makeConfig(t, 5, 7, 3, []int{0, 0, 0, 0, 1, 1, 1}, true)

	randomServer := int(nrand(5))
	cfg.crashCounter(randomServer)
	cfg.crashCounter((randomServer + 1) % 5)
	cfg.crashCounter((randomServer + 2) % 5)

	cfg.startVoting()

	voteNoResult := cfg.voteResult()
	if voteNoResult != -1 {
		cfg.t.Fatalf("expecting no result, but got %v", voteNoResult)
	}

	cfg.startCounter((randomServer + 1) % 5)
	cfg.connectCounter((randomServer + 1) % 5)

	voteResult := cfg.voteResult()
	if voteResult != 0 {
		cfg.t.Fatalf("expecting 0, but got %v", voteResult)
	} else {
		fmt.Println("ok")
	}

	cfg.cleanup()
}
