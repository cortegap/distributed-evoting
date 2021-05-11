package election

import (
	"sync"
	"sync/atomic"
	"time"

	"6.824/labrpc"
)

const receiveTotalsTimeout int32 = 500
const votingTimeout int32 = 10000

// Possible states for a server during election day
type Stage int

const (
	Open   Stage = 1
	Closed Stage = 0
)

type VoteCounter struct {
	mu    sync.Mutex
	dead  int32 // set by Kill()
	stage Stage

	committeeMembers []*labrpc.ClientEnd
	me               int
	nCounters        int
	nVoters          int
	nShares          int

	votes          map[int64]int64
	winner         int
	totalCounts    map[int]int64   // Holds sum of all the cm's
	votesToCount   map[int64]int64 // Subset of votes to be counted
	countingClique map[int]bool    // CMs in clique to count votes
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

type SendIdsArgs struct {
	Server int
	Ids    map[int64]bool
}

type SendIdsReply struct {
	Success bool
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

		if len(vc.votes) == vc.nVoters {
			vc.stage = Closed
			go vc.ExchangeVoters()
		}
	}
}

func (vc *VoteCounter) ExchangeVoters() {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	for k, v := range vc.votes {
		vc.votesToCount[k] = v
	}

	vc.countingClique[vc.me] = true

	for i := 0; i < len(vc.committeeMembers); i++ {
		if i != vc.me {
			ids := make(map[int64]bool)
			for k := range vc.votes {
				ids[k] = true
			}

			go func(counter int, me int, voters map[int64]bool) {
				args := SendIdsArgs{me, voters}
				reply := SendIdsReply{}
				vc.sendIdList(counter, &args, &reply)

			}(i, vc.me, ids)
		}
	}

	for len(vc.countingClique) < vc.nShares {
		time.Sleep(time.Duration(receiveTotalsTimeout) * time.Millisecond)
	}

	go vc.ProcessElection()
}

func (vc *VoteCounter) sendIdList(counter int, args *SendIdsArgs, reply *SendIdsReply) {
	if !vc.killed() {
		vc.committeeMembers[counter].Call("VoteCounter.IdList", args, reply)
	}
}

func (vc *VoteCounter) IdList(args *SendIdsArgs, reply *SendIdsReply) {
	if !vc.killed() {
		_, ok := vc.countingClique[args.Server]
		if !ok {
			vc.countingClique[args.Server] = true
			toDelete := []int64{}
			for id := range vc.votesToCount {
				if _, ok2 := args.Ids[id]; !ok2 {
					toDelete = append(toDelete, id)
				}
			}
			for _, id := range toDelete {
				delete(vc.votesToCount, id)
			}
		}
	}
}

//
// Count the votes, and announce the winner (not fault tolerant)
//
func (vc *VoteCounter) ProcessElection() {
	// Add the votes that should be counted
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
// Add all of the votes
//
func (vc *VoteCounter) AddVotes() int64 {
	if !vc.killed() {
		vc.mu.Lock()
		defer vc.mu.Unlock()

		vc.totalCounts[vc.me+1] = 0
		for _, val := range vc.votesToCount {
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
	if !vc.killed() {
		vc.mu.Lock()
		defer vc.mu.Unlock()

		if _, ok := vc.countingClique[args.Index]; ok {
			vc.totalCounts[args.Index] = args.Value
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

		if int(totalVotes) > len(vc.votesToCount)/2 {
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

	if vc.winner != -1 {
		done = true
		winner = vc.winner
	}
	return done, winner
}

func (vc *VoteCounter) electionTimeout() {
	time.Sleep(time.Duration(votingTimeout) * time.Millisecond)

	if !vc.killed() {
		vc.mu.Lock()
		if vc.stage == Open {
			vc.stage = Closed
			go vc.ExchangeVoters()
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

//
// main/votecounter.go calls this function.
//
func MakeVoteCounter(committeeMembers []*labrpc.ClientEnd, me, nCounters, nVoters, nShares int) *VoteCounter {
	vc := &VoteCounter{}

	vc.committeeMembers = committeeMembers
	vc.me = me
	vc.stage = Open
	vc.nCounters = nCounters
	vc.nVoters = nVoters
	vc.votes = make(map[int64]int64)
	vc.votesToCount = make(map[int64]int64)
	vc.countingClique = make(map[int]bool)
	vc.nShares = nShares
	vc.totalCounts = make(map[int]int64)
	vc.winner = -1

	go vc.electionTimeout()

	return vc
}
