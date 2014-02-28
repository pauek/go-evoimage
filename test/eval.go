package main

import (
	"flag"
	"fmt"
	eimg "go-evoimage"
	"image/png"
	"os"
	// "runtime"
)

var (
	Size int
	Samples int
)

func main() {
	// runtime.GOMAXPROCS(runtime.NumCPU())
	flag.IntVar(&Size, "s", 500, "Image size")
	flag.IntVar(&Samples, "k", 4, "Image size")
	flag.Parse()
	args := flag.Args()
	e, err := eimg.Read(args[0])
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	fmt.Println(e)
	img := e.Render(Size, Samples)
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
