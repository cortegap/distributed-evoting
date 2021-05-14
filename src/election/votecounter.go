package election

import (
	"sync"
	"sync/atomic"
	"time"

	"6.824/labrpc"
)

const receiveTotalsTimeout int32 = 500
const votingTimeout int32 = 3000
const exchangeTimeout int32 = 3000

//
// Add all of the votes
//
func addVotes(votes map[int64]int64) (votesSum int64) {
	for _, val := range votes {
		votesSum += val
	}

	votesSum %= field

	return
}

//	Compute the winner of the election
func computeWinner(shares map[int]int64, nVoters int) int {
	total := float64(0)
	for xi, yi := range shares {
		partial := float64(yi)
		for x := range shares {
			if xi != x {
				den := x - xi
				num := x
				var fraction float64 = float64(num) / float64(den)
				partial *= fraction
			}
		}
		total += partial
	}

	totalVotes := int64(total) % field

	if totalVotes < 0 {
		totalVotes += field
	}

	if int(totalVotes) > nVoters/2 {
		return 1
	} else {
		return 0
	}
}

type CountTotalArgs struct {
	Index int
	Value int64
}

type CountTotalReply struct {
	Success bool
}

type VoteCounter struct {
	mu   sync.Mutex
	dead int32 // set by Kill()

	committeeMembers []*labrpc.ClientEnd
	me               int

	votes             map[int64]int64
	nVoters           int
	submissionSuccess map[int]bool // TODO: Decide on data structure

	totalCounts map[int]int64 // Holds sum of all the cm's
	threshold   int
	winner      int
}

//
// main/votecounter.go calls this function.
//
func MakeVoteCounter(committeeMembers []*labrpc.ClientEnd, me, nVoters, threshold int) *VoteCounter {
	vc := &VoteCounter{}

	vc.committeeMembers = committeeMembers
	vc.me = me

	vc.votes = make(map[int64]int64)
	vc.nVoters = nVoters
	vc.submissionSuccess = make(map[int]bool)

	vc.totalCounts = make(map[int]int64)
	vc.threshold = threshold
	vc.winner = -1

	return vc
}

func (vc *VoteCounter) CountVote(args *CountVoteArgs, reply *CountVoteReply) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.votes[args.VoterId] = args.Vote
	reply.Success = true

	_, shareTotalSent := vc.totalCounts[vc.me+1]
	if len(vc.votes) == vc.nVoters && !shareTotalSent {
		vc.totalCounts[vc.me+1] = addVotes(vc.votes)

		go vc.sendShareTotal()
	}
}

//
// Count the votes, and announce the winner (not fault tolerant)
//
func (vc *VoteCounter) sendShareTotal() {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	for !vc.killed() && len(vc.submissionSuccess) < len(vc.committeeMembers)-1 {
		for i := 0; i < len(vc.committeeMembers); i++ {
			_, alreadySubmitted := vc.submissionSuccess[i]
			if i != vc.me && !alreadySubmitted {
				go func(counter, index int, total int64) {
					args := CountTotalArgs{index, total}
					reply := CountTotalReply{}
					vc.sendCountTotal(counter, &args, &reply)
				}(i, vc.me+1, vc.totalCounts[vc.me+1])
			}
		}

		vc.mu.Unlock()
		time.Sleep(time.Duration(receiveTotalsTimeout) * time.Millisecond)
		vc.mu.Lock()
	}
}

func (vc *VoteCounter) sendCountTotal(counter int, args *CountTotalArgs, reply *CountTotalReply) {
	ok := vc.committeeMembers[counter].Call("VoteCounter.CountTotal", args, reply)

	if ok && reply.Success {
		vc.mu.Lock()
		vc.submissionSuccess[counter] = true
		vc.mu.Unlock()
	}
}

//
//	Get the total of other vote counters
//
func (vc *VoteCounter) CountTotal(args *CountTotalArgs, reply *CountTotalReply) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.totalCounts[args.Index] = args.Value
	reply.Success = true

	if len(vc.totalCounts) >= vc.threshold {
		vc.winner = computeWinner(vc.totalCounts, vc.nVoters)
	}
}

//
// Returns whether or not there is a winner and
// the winner. If there is no winner, returns -1.
//
func (vc *VoteCounter) Done() (bool, int) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	return vc.winner != -1, vc.winner
}

//
// the tester doesn't halt goroutines created after each test,
// but it does call the Kill() method. The use of atomic avoids the
// need for a lock.
//
func (vc *VoteCounter) Kill() {
	atomic.StoreInt32(&vc.dead, 1)
}

//
// Check whether Kill() has been called.
//
func (vc *VoteCounter) killed() bool {
	z := atomic.LoadInt32(&vc.dead)
	return z == 1
}
