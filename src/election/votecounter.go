package election

import (
	"sync"

	"6.824/labrpc"
)

type VoteCounter struct {
	mu               sync.Mutex
	committeeMembers []*labrpc.ClientEnd
	me               int
	nVoters          int
	votes            map[int64]int
}

type CountVoteArgs struct {
	VoterId int64
	Vote    int
}

type CountVoteReply struct {
	Success bool
}

func (vc *VoteCounter) CountVote(args *CountVoteArgs, reply *CountVoteReply) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	_, ok := vc.votes[args.VoterId]
	if len(vc.votes) == vc.nVoters && !ok {
		reply.Success = false
		return
	}

	vc.votes[args.VoterId] = args.Vote
}

func (vc *VoteCounter) Done() bool {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	return len(vc.votes) >= vc.nVoters
}

//
// main/votecounter.go calls this function.
//
func MakeVoteCounter(committeeMembers []*labrpc.ClientEnd, me, nVoters int) *VoteCounter {
	vc := &VoteCounter{}

	vc.committeeMembers = committeeMembers
	vc.me = me
	vc.nVoters = nVoters
	vc.votes = make(map[int64]int)

	return vc
}
