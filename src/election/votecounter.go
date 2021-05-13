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

// Possible states for a server during election day
type Stage int

const (
	Open     Stage = 0
	Ready    Stage = 1
	Counting Stage = 2
	Done     Stage = 3
	Failed   Stage = -1
)

type SendTotalArgs struct {
	Index int
	Value int64
}

type SendTotalReply struct {
	Success bool
}

type VoteCounter struct {
	mu    sync.Mutex
	dead  int32 // set by Kill()
	stage Stage

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

	vc.stage = Open

	vc.committeeMembers = committeeMembers
	vc.me = me
	vc.nVoters = nVoters
	vc.threshold = threshold

	vc.votes = make(map[int64]int64)
	vc.winner = -1
	vc.totalCounts = make(map[int]int64)

	go vc.electionTimeout()

	return vc
}

func (vc *VoteCounter) electionTimeout() {
	time.Sleep(time.Duration(votingTimeout) * time.Millisecond)

	if !vc.killed() {

		vc.mu.Lock()
		if vc.stage == Ready {
			go vc.ProcessElection()
			go vc.exchangeTimeout()
		} else {
			vc.stage = Failed
		}
		vc.mu.Unlock()
	}
}

func (vc *VoteCounter) CountVote(args *CountVoteArgs, reply *CountVoteReply) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	if !vc.killed() && vc.stage == Open {

		_, ok := vc.votes[args.VoterId]
		if len(vc.votes) == vc.nVoters && !ok {
			reply.Success = false
			return
		}

		vc.votes[args.VoterId] = args.Vote
		reply.Success = true

		if len(vc.votes) == vc.nVoters {
			vc.stage = Ready
		}
	}
}

//
// Count the votes, and announce the winner (not fault tolerant)
//
func (vc *VoteCounter) ProcessElection() {
	// Add the votes that should be counted
	vc.AddVotes()

	vc.mu.Lock()
	for vc.stage == Ready { // Send sum to other CMs
		vc.mu.Unlock()
		for i := 0; i < len(vc.committeeMembers); i++ {
			if i != vc.me {
				go func(counter int, index int, total int64) {
					args := SendTotalArgs{index, total}
					reply := SendTotalReply{}
					vc.sendShareTotal(counter, &args, &reply)

				}(i, vc.me+1, vc.totalCounts[vc.me+1])
			}
		}

		time.Sleep(time.Duration(receiveTotalsTimeout) * time.Millisecond)
		vc.mu.Lock()
	}
	vc.mu.Unlock()
}

//
// Add all of the votes
//
func (vc *VoteCounter) AddVotes() int64 {
	if !vc.killed() {
		vc.mu.Lock()
		defer vc.mu.Unlock()

		vc.totalCounts[vc.me+1] = 0
		for _, val := range vc.votes {
			vc.totalCounts[vc.me+1] += val
		}

		vc.totalCounts[vc.me+1] %= field

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
	if !vc.killed() && vc.stage == Ready {

		vc.mu.Lock()
		vc.totalCounts[args.Index] = args.Value
		vc.mu.Unlock()

		if done, _ := vc.Done(); !done && len(vc.totalCounts) >= vc.threshold {
			go vc.computeWinner()
		}
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

		totalVotes := int64(total) % field

		if totalVotes < 0 {
			totalVotes += field
		}

		if int(totalVotes) > vc.nVoters/2 {
			vc.winner = 1
		} else {
			vc.winner = 0
		}
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

	if vc.stage == Failed {
		return true, -1
	}

	if vc.winner != -1 {
		done = true
		winner = vc.winner
	}

	return done, winner
}

func (vc *VoteCounter) exchangeTimeout() {
	time.Sleep(time.Duration(exchangeTimeout) * time.Millisecond)

	if !vc.killed() {
		done, _ := vc.Done()
		vc.mu.Lock()

		if len(vc.totalCounts) >= vc.threshold {
			if !done {
				go vc.computeWinner()
			}
			vc.stage = Counting
		} else {
			vc.stage = Failed
		}
		vc.mu.Unlock()
	}
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
