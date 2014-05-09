package main

import (
	"flag"
	"fmt"
	eimg "go-evoimage"
	"math/rand"
	"time"
)

var (
	Seed       int64
	NumMutants int
	NumNodes   int
)

func main() {
	flag.Int64Var(&Seed, "s", 0, "Seed")
	flag.IntVar(&NumNodes, "k", 5, "Number of nodes in random module")
	flag.IntVar(&NumMutants, "n", 10, "Number of mutants to generate")
	flag.Parse()

	if Seed == 0 {
		Seed = time.Now().UnixNano()
	}
	rand.Seed(Seed)

	e := eimg.RandomCircuit(NumNodes)
	fmt.Println("   ", e)
	for i := 0; i < NumMutants; i++ {
		c := e.Clone()
		c.Mutate()
		fmt.Println(c)
	}
}
