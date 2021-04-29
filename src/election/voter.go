package election

import (
	"crypto/rand"
	"math/big"
	"sync"

	"6.824/labrpc"
)

type Voter struct {
	mu               sync.Mutex
	committeeMembers []*labrpc.ClientEnd
	voterId          int64
	vote             int
	nShares          int
	shares           []int
	done             bool
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

//
// Function to create Shamir Shares
//
func (vt *Voter) makeShares() {
	return
}

func (vt *Voter) SendShares() {
	for !vt.Done() {
		return
	}
}

func (vt *Voter) Done() bool {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	return vt.done
}

//
// main/voter.go calls this function.
//
func MakeVoter(committeeMembers []*labrpc.ClientEnd, vote int, nShares int) *Voter {
	vt := &Voter{}

	vt.committeeMembers = committeeMembers
	vt.nShares = nShares
	vt.voterId = nrand()
	vt.vote = vote
	vt.shares = make([]int, len(committeeMembers))

	go vt.makeShares()

	return vt
}
