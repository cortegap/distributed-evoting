package election

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"6.824/labgob"
	"6.824/labrpc"
)

// TODO: tune this
const voterTimeout int32 = 100
const field int64 = 1104637706180507

func nrand(max int64) int64 {
	bigmax := big.NewInt(max)
	if max == 0 {
		bigmax = big.NewInt(int64(1) << 62)
	}
	bigx, _ := rand.Int(rand.Reader, bigmax)
	x := bigx.Int64()
	return x
}

// Evaluate polynomial
// f(x) = coef[0] + coef[0]x + coef[0]x^2 + ...
func evalPolynimialL(coef []int64, x int64) int64 {
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

type CountVoteArgs struct {
	VoterId int64
	Vote    int64
}

type CountVoteReply struct {
	Success bool
}

type Voter struct {
	mu   sync.Mutex
	dead int32

	voterId            int64
	persister          Persister
	committeeMembers   []*labrpc.ClientEnd
	submissionSuccess  []bool
	nSubmissionSuccess int

	vote      int
	shares    []int64
	threshold int
}

type Persister interface {
	readPersistState() []byte
	writePersistState([]byte)
}

//
// main/voter.go calls this function.
//
func MakeVoter(committeeMembers []*labrpc.ClientEnd, vote, threshold int, persister Persister) *Voter {
	vt := &Voter{}

	vt.voterId = nrand(0)
	vt.persister = persister
	vt.committeeMembers = committeeMembers
	vt.submissionSuccess = make([]bool, len(committeeMembers))

	vt.vote = vote
	vt.shares = make([]int64, len(committeeMembers))
	vt.threshold = threshold

	if !vt.readPersist() {
		vt.makeShares()
		vt.persist()
	}

	return vt
}

func (vt *Voter) readPersist() bool {
	data := vt.persister.readPersistState()

	if data == nil || len(data) < 1 { // bootstrap without any state?
		return false
	}

	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var vote int
	var shares []int64

	if d.Decode(&vote) != nil || d.Decode(&shares) != nil {
		panic("Error decoding persist data")
	} else if vt.vote != vote {
		panic("Error decoding persist data")
	} else {
		vt.shares = shares
	}

	return true
}

func (vt *Voter) persist() {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(vt.vote)
	e.Encode(vt.shares)
	data := w.Bytes()
	vt.persister.writePersistState(data)
}

//
// Function to create Shamir Shares
// - Polynomial of degree nShares over Z_field
// - len(vt.committeeMembers) shares
//
func (vt *Voter) makeShares() {
	// Polynomial of degree nShares-1 with coefficients in field Z_field
	coefficients := make([]int64, vt.threshold)

	coefficients[0] = int64(vt.vote)
	for i := 1; i < vt.threshold; i++ {
		val, _ := rand.Int(rand.Reader, big.NewInt(field))
		coefficients[i] = val.Int64()
	}

	// Compute shares. For each committe member i, evaluate polynomial
	// at x = i + 1
	for i := range vt.shares {
		vt.shares[i] = evalPolynimialL(coefficients, int64(i+1))
	}
}

//
// Send votes to the vote counter until the end
//
func (vt *Voter) Vote() {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	for !vt.killed() && vt.nSubmissionSuccess < len(vt.shares) {
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
		vt.mu.Lock()
	}
}

//
//  Send Count Votes RPC
//
func (vt *Voter) sendCountVote(counter int, args *CountVoteArgs, reply *CountVoteReply) {
	ok := vt.committeeMembers[counter].Call("VoteCounter.CountVote", args, reply)

	if ok && reply.Success {
		// Update submissionSuccess as done for this server
		vt.mu.Lock()
		if !vt.submissionSuccess[counter] {
			vt.submissionSuccess[counter] = true
			vt.nSubmissionSuccess++
		}
		vt.mu.Unlock()
	}
}

func (vt *Voter) Kill() {
	atomic.StoreInt32(&vt.dead, 1)
}

func (vt *Voter) killed() bool {
	z := atomic.LoadInt32(&vt.dead)
	return z == 1
}

func (vt *Voter) Done() bool {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	return vt.nSubmissionSuccess == len(vt.shares)
}
