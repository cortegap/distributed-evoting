package main

//
// start the a voter, passing your vote as an argument
//
// go run voter.go 1
//

import (
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: voter args...\n")
		os.Exit(1)
	}

	time.Sleep(time.Second)
}
