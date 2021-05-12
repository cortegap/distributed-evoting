package election

import (
	"crypto/rand"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"6.824/labrpc"
)

// TODO: tune this
const voterTimeout int32 = 1000

const field int64 = 1104637706180507

type Voter struct {
	mu   sync.Mutex
	done int32

	committeeMembers  []*labrpc.ClientEnd
	submissionSuccess []bool
	voterId           int64
	vote              int

	nShares int
	shares  []int64
}

func nrand(max int64) int64 {
	bigmax := big.NewInt(max)
	if max == 0 {
		bigmax = big.NewInt(int64(1) << 62)
	}
	bigx, _ := rand.Int(rand.Reader, bigmax)
	x := bigx.Int64()
	return x
}

//
// Function to create Shamir Shares
// - Polynomial of degree vt.nShares over Z_field
// - len(vt.committeeMembers) shares
//
func (vt *Voter) makeShares() {
	if !vt.killed() {
		vt.mu.Lock()
		defer vt.mu.Unlock()

		// Polynomial of degree nShares-1 with coefficients in field Z_field
		coefficients := make([]int64, vt.nShares)

		coefficients[0] = int64(vt.vote)
		for i := 1; i < vt.nShares; i++ {
			val, _ := rand.Int(rand.Reader, big.NewInt(field))
			coefficients[i] = val.Int64()
		}

		// Compute shares. For each committe member i, evaluate polynomial
		// at x = i + 1
		for i := range vt.shares {
			vt.shares[i] = vt.evalPolynimialL(coefficients, int64(i+1))
		}
	}
}

//
// Evaluate polynomial
// f(x) = coef[0] + coef[0]x + coef[0]x^2 + ...
//
func (vt *Voter) evalPolynimialL(coef []int64, x int64) int64 {
	fx := big.NewInt(0)
	bigX := big.NewInt(x)

	for exp, c := range coef {
		var monomial big.Int
		monomial = *monomial.Exp(bigX, big.NewInt(int64(exp)), big.NewInt(0)) // x^exp
		monomial.Mul(&monomial, big.NewInt(c))
		fx.Add(fx, &monomial)
	}

	fx.Mod(fx, big.NewInt(field))
	return fx.Int64()
}

//
// Send votes to the vote counter until the end
//
func (vt *Voter) SendShares() {
	vt.makeShares()

	for !vt.killed() {
		vt.mu.Lock()

		// Send CountVote RPCs to everyone
		for i := 0; i < len(vt.committeeMembers); i++ {
			if !vt.submissionSuccess[i] {
				go func(id int64, vote int64, counter int) {
					args := CountVoteArgs{id, vote}
					reply := CountVoteReply{}
					vt.sendCountVote(counter, &args, &reply)

				}(vt.voterId, vt.shares[i], i)
			}
		}

		vt.mu.Unlock()
		time.Sleep(time.Duration(voterTimeout) * time.Millisecond)
	}
}

//
//  Send Count Votes RPC
//
func (vt *Voter) sendCountVote(counter int, args *CountVoteArgs, reply *CountVoteReply) {
	if !vt.killed() {
		ok := vt.committeeMembers[counter].Call("VoteCounter.CountVote", args, reply)

		if ok && reply.Success {
			// Update submissionSuccess as done for this server
			vt.mu.Lock()
			vt.submissionSuccess[counter] = true

			// Check if we finished
			count := 0
			for _, done := range vt.submissionSuccess {
				if done {
					count++
				} else {
					break
				}
			}

			if count == len(vt.committeeMembers) {
				vt.done = 1
				vt.mu.Unlock()
				vt.Kill()
			} else {
				vt.mu.Unlock()
			}
		}
	}
}

//
// The tester doesn't halt goroutines created after each test,
// but it does call the Kill() method. The use of atomic avoids the
// need for a lock.
//
func (vt *Voter) Kill() {
	atomic.StoreInt32(&vt.done, 1)
}

//
// Check whether Kill() has been called.
//
func (vt *Voter) killed() bool {
	z := atomic.LoadInt32(&vt.done)
	return z == 1
}

func (vt *Voter) Done() bool {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	return vt.done == 1
}

//
// main/voter.go calls this function.
//
func MakeVoter(committeeMembers []*labrpc.ClientEnd, nShares, vote int) *Voter {
	vt := &Voter{}

	vt.committeeMembers = committeeMembers
	vt.nShares = nShares
	vt.voterId = nrand(0)
	vt.vote = vote
	vt.shares = make([]int64, len(committeeMembers))
	vt.submissionSuccess = make([]bool, len(committeeMembers))
	vt.done = 0

	go vt.SendShares()

	return vt
}
