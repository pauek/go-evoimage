package main

import (
	"flag"
	"fmt"
	eimg "go-evoimage"
	"math/rand"
	"time"
)

var (
	Seed        int64
	NumCircuits int
	NumNodes    int
)

func main() {
	flag.Int64Var(&Seed, "s", 0, "Seed")
	flag.IntVar(&NumCircuits, "n", 1, "Number of circuits to generate")
	flag.IntVar(&NumNodes, "k", 5, "Number of nodes in random module")
	flag.Parse()

	if Seed == 0 {
		Seed = time.Now().UnixNano()
	}
	rand.Seed(Seed)
	for i := 0; i < NumCircuits; i++ {
		e := eimg.RandomCircuit(NumNodes)
		fmt.Println(e)
	}
}
