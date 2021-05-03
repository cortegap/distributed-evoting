package election

import (
	"sync"
	"sync/atomic"
	"time"

	"6.824/labrpc"
)

const receiveTotalsTimeout int32 = 500

type VoteCounter struct {
	mu               sync.Mutex
	committeeMembers []*labrpc.ClientEnd
	me               int
	nCounters        int
	nVoters          int
	votes            map[int64]int64
	nShares          int
	dead             int32 // set by Kill()
	winner           int
	totalCounts      map[int]int64
}

type CountVoteArgs struct {
	VoterId int64
	Vote    int64
}

type CountVoteReply struct {
	Success bool
}

type SendTotalArgs struct {
	Index int
	Value int64
}

type SendTotalReply struct {
	Success bool
}

func (vc *VoteCounter) CountVote(args *CountVoteArgs, reply *CountVoteReply) {
	if !vc.killed() {
		vc.mu.Lock()
		defer vc.mu.Unlock()

		_, ok := vc.votes[args.VoterId]
		if len(vc.votes) == vc.nVoters && !ok {
			reply.Success = false
			return
		}

		vc.votes[args.VoterId] = args.Vote

		if len(vc.votes) == vc.nVoters {
			go vc.ProcessElection()
		}
	}
}

//
// Close election, count the votes, and announce the winner (not fault tolerant)
//
func (vc *VoteCounter) ProcessElection() {
	vc.AddVotes()

	for i := 0; i < len(vc.committeeMembers); i++ {
		go func(counter int, index int, total int64) {
			args := SendTotalArgs{index, total}
			reply := SendTotalReply{}
			vc.sendShareTotal(counter, &args, &reply)

		}(i, vc.me+1, vc.totalCounts[vc.me+1])
	}

	for len(vc.totalCounts) < vc.nShares {
		time.Sleep(time.Duration(receiveTotalsTimeout) * time.Millisecond)
	}

	vc.computeWinner()
}

//
// Add all of the votes (not fault tolerant)
//
func (vc *VoteCounter) AddVotes() int64 {
	if !vc.killed() {
		vc.mu.Lock()
		defer vc.mu.Unlock()

		vc.totalCounts[vc.me+1] = 0
		for _, val := range vc.votes {
			vc.totalCounts[vc.me+1] += val
		}

		return vc.totalCounts[vc.me+1]
	}

	return 0
}

//
// Send RPC with the sum of votes to the other vote counters
// (not fault tolerant)
//
func (vc *VoteCounter) sendShareTotal(counter int, args *SendTotalArgs, reply *SendTotalReply) {
	if !vc.killed() {
		vc.committeeMembers[counter].Call("VoteCounter.ShareTotal", args, reply)
	}
}

//
//	Get the total of other vote counters
//
func (vc *VoteCounter) ShareTotal(args *SendTotalArgs, reply *SendTotalReply) {
	if !vc.killed() {
		vc.mu.Lock()
		defer vc.mu.Unlock()

		vc.totalCounts[args.Index] = args.Value
	}
}

//
//	Compute the winner of the election
//
func (vc *VoteCounter) computeWinner() {
	if !vc.killed() {
		vc.mu.Lock()
		defer vc.mu.Unlock()

		total := float64(0)
		for xi, yi := range vc.totalCounts {
			partial := float64(yi)
			for x := range vc.totalCounts {
				if xi != x {
					den := x - xi
					num := x
					var fraction float64 = float64(num) / float64(den)
					partial *= fraction
				}
			}
			total += partial
		}

		vc.winner = int(int64(total) % field)
	}
}

//
// Returns whether or not there is a winner and
// the winner. If there is no winner, returns -1.
//
func (vc *VoteCounter) Done() (bool, int) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	done := false
	winner := -1

	if vc.winner != -1 {
		done = true
		winner = vc.winner
	}
	return done, winner
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

//
// main/votecounter.go calls this function.
//
func MakeVoteCounter(committeeMembers []*labrpc.ClientEnd, me, nCounters, nVoters, nShares int) *VoteCounter {
	vc := &VoteCounter{}

	vc.committeeMembers = committeeMembers
	vc.me = me
	vc.nCounters = nCounters
	vc.nVoters = nVoters
	vc.votes = make(map[int64]int64)
	vc.nShares = nShares
	vc.totalCounts = make(map[int]int64)
	vc.winner = -1

	return vc
}
