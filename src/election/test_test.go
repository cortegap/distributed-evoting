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
			done, vote := cfg.counters[i].Done()
			if done {
				return vote
			}
		}
	}
	return -1
}

func testSmallElection(t *testing.T, description string, unreliable bool) {
	fmt.Println(description)
	cfg := makeConfig(t, 3, 5, []int{0, 0, 0, 1, 1}, unreliable)

	voteResult := cfg.voteResult()
	if voteResult != 0 {
		cfg.t.Fatalf("expecting result of 0, but got %v", voteResult)
	} else {
		fmt.Println("ok")
	}
}

func TestInitialElection0(t *testing.T) {
	fmt.Println("Starting simple test - 0 wins")
	cfg := makeConfig(t, 3, 5, []int{0, 0, 0, 1, 1}, false)

	voteResult := cfg.voteResult()
	if voteResult != 0 {
		cfg.t.Fatalf("expecting result of 0, but got %v", voteResult)
	} else {
		fmt.Println("ok")
	}
}

func TestInitialElection1(t *testing.T) {
	fmt.Println("Starting simple test - 1 wins")
	cfg := makeConfig(t, 3, 5, []int{0, 0, 1, 1, 1}, false)

	voteResult := cfg.voteResult()
	if voteResult != 1 {
		cfg.t.Fatalf("expecting result of 1, but got %v", voteResult)
	} else {
		fmt.Println("ok")
	}
}

func TestUnreliableElection0(t *testing.T) {
	fmt.Println("Starting unreliable election test - 0 wins")
	cfg := makeConfig(t, 3, 5, []int{0, 0, 0, 1, 1}, true)

	voteResult := cfg.voteResult()
	if voteResult != 0 {
		cfg.t.Fatalf("expecting result of 0, but got %v", voteResult)
	} else {
		fmt.Println("ok")
	}
}

func TestUnreliableElection1(t *testing.T) {
	fmt.Println("Starting unreliable election test - 1 wins")
	cfg := makeConfig(t, 3, 5, []int{0, 0, 1, 1, 1}, true)

	voteResult := cfg.voteResult()
	if voteResult != 1 {
		cfg.t.Fatalf("expecting result of 1, but got %v", voteResult)
	} else {
		fmt.Println("ok")
	}
}
