package main

import (
	"bufio"
	"flag"
	"fmt"
	eimg "go-evoimage"
	"os"
	"os/exec"
	"sync"
)

var (
	Print bool
	curr  = 1
	wg    sync.WaitGroup
)

func Graphviz(n int, expr string) {
	// Read expression from line
	e, err := eimg.Read(expr)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}

	if Print {
		e.Graphviz(os.Stdout)
		return
	}

	// Write dot file
	dotfile := fmt.Sprintf("img%04d.dot", n)
	file, err := os.Create(dotfile)
	if err != nil {
		fmt.Printf("ERROR: Cannot create '%s': %s", dotfile, err)
		os.Exit(1)
	}
	e.Graphviz(file)
	file.Close()

	// invoke dot
	pngfile := fmt.Sprintf("img%04dg.png", n)
	dot := exec.Command("dot", "-Tpng", "-o", pngfile, dotfile)
	if err := dot.Run(); err != nil {
		fmt.Printf("ERROR: Cannot run 'dot -Tpng -o %s %s': %s",
			pngfile, dotfile, err)
	}

	// remove dot file
	if err := os.Remove(dotfile); err != nil {
		fmt.Printf("ERROR: Cannot delete file '%s': %s", dotfile, err)
	}

	fmt.Println(e)
	wg.Done()
}

func main() {
	flag.BoolVar(&Print, "p", false, "Show the file on stdout")
	flag.Parse()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		wg.Add(1)
		go Graphviz(curr, scanner.Text())
		curr++
	}
	wg.Wait()
}
