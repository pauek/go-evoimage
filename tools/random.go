package main

import (
	"flag"
	"fmt"
	eimg "go-evoimage"
	"math/rand"
	"runtime"
	"time"
)

var (
	NumCircuits int
	NumNodes    int
)

func main() {
	rand.Seed(time.Now().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.IntVar(&NumCircuits, "n", 1, "Number of circuits to generate")
	flag.IntVar(&NumNodes, "k", 5, "Number of nodes in random module")
	flag.Parse()

	for i := 0; i < NumCircuits; i++ {
		e := eimg.RandomCircuit(NumNodes)
		fmt.Println(e)
	}
}
