package main

import (
	"bufio"
	"flag"
	"fmt"
	eimg "go-evoimage"
	"os"
	"os/exec"
)

var (
	Print bool
)

func main() {
	flag.BoolVar(&Print, "p", false, "Show the file on stdout")
	flag.Parse()

	scanner := bufio.NewScanner(os.Stdin)
	curr := 1
	for scanner.Scan() {
		// Read expression from line
		e, err := eimg.Read(scanner.Text())
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		if Print {
			e.Graphviz(os.Stdout)
			continue
		}

		// Write dot file
		dotfile := fmt.Sprintf("img%04d.dot", curr)
		file, err := os.Create(dotfile)
		if err != nil {
			fmt.Printf("ERROR: Cannot create '%s': %s", dotfile, err)
			os.Exit(1)
		}
		e.Graphviz(file)
		file.Close()

		// invoke dot
		pngfile := fmt.Sprintf("img%04dg.png", curr)
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

		curr++
	}
}
