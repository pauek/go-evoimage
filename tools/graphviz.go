package main

import (
	"bufio"
	"flag"
	"fmt"
	eimg "go-evoimage"
	"os"
)

var (
	NumNodes int
)

func main() {
	flag.IntVar(&NumNodes, "n", 5, "Number of nodes in random module")
	flag.Parse()

	scanner := bufio.NewScanner(os.Stdin)
	curr := 1
	for scanner.Scan() {
		e, err := eimg.Read(scanner.Text())
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}
		filename := fmt.Sprintf("graph%04d.dot", curr)
		curr++
		file, err := os.Create(filename)
		if err != nil {
			fmt.Printf("ERROR: Cannot create '%s': %s", filename, err)
			os.Exit(1)
		}
		e.Graphviz(file)
		file.Close()
	}
}
