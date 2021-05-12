package main

//
// start the a voter, passing your vote as an argument
//
// go run voter.go 1
//

// import (
// 	"fmt"
// 	"os"
// 	"strconv"
// 	"time"

// 	"6.824/election"
// 	"6.824/labrpc"
// )

// func main() {
// 	if len(os.Args) < 2 {
// 		fmt.Fprintf(os.Stderr, "Usage: voter args...\n")
// 		os.Exit(1)
// 	}

// 	vote, err := strconv.Atoi(os.Args[1])
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Usage: voter arg must be an int...\n")
// 		os.Exit(1)
// 	}

// 	vt := election.MakeVoter(make([]*labrpc.ClientEnd, 1), vote, 3, 7)
// 	for vt.Done() == false {
// 		time.Sleep(time.Second)
// 	}

// 	time.Sleep(time.Second)
// }
