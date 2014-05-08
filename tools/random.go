package main

import (
	"flag"
	"fmt"
	eimg "go-evoimage"
	"image/png"
	"math/rand"
	"os"
	"runtime"
	"time"
)

var (
	NumNodes  int
	PrintOnly bool
	Graphviz  bool
)

func main() {
	rand.Seed(time.Now().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.IntVar(&NumNodes, "n", 5, "Number of nodes in random module")
	flag.BoolVar(&PrintOnly, "p", false, "Only Print expression")
	flag.BoolVar(&Graphviz, "g", false, "Print Graphviz")
	flag.Parse()
	e := eimg.RandomCircuit(NumNodes)
	if Graphviz {
		e.Graphviz(os.Stdout)
		fmt.Fprintln(os.Stderr, e)
	} else {
		fmt.Println(e)
	}

	if !PrintOnly {
		img := e.Render(500, 5)
		f, err := os.Create("img.png")
		if err != nil {
			fmt.Printf("Cannot open 'img.png': %s", err)
			os.Exit(1)
		}
		err = png.Encode(f, img)
		if err != nil {
			fmt.Printf("Cannot encode PNG: %s", err)
			os.Exit(1)
		}
	}
}
