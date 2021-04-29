package main

//
// start a vote counting process, passing the number of
// committee members (vote counters) for the election as an argument
//
// go run votecounter.go 3
//

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"6.824/election"
	"6.824/labrpc"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: votecounter args...\n")
		os.Exit(1)
	}

	nVoters, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Usage: votecounter arg must be an int...\n")
		os.Exit(1)
	}

	vc := election.MakeVoteCounter(make([]*labrpc.ClientEnd, 1), 0, nVoters)
	for vc.Done() == false {
		time.Sleep(time.Second)
	}

	time.Sleep(time.Second)
}
