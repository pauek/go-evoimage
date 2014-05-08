package main

import (
	"bufio"
	"flag"
	"fmt"
	eimg "go-evoimage"
	"image/png"
	"os"
	"runtime"
)

var (
	Size    int
	Samples int
	Curr    int = 1
)

func render(expr string) {
	e, err := eimg.Read(expr)
	if err != nil {
		fmt.Println("ERROR: ", err)
		os.Exit(1)
	}
	fmt.Println(e)
	img := e.Render(Size, Samples)
	imgname := fmt.Sprintf("img%04d.png", Curr)
	Curr++
	f, err := os.Create(imgname)
	if err != nil {
		fmt.Printf("Cannot open '%s': %s", imgname, err)
		os.Exit(1)
	}
	err = png.Encode(f, img)
	if err != nil {
		fmt.Printf("Cannot encode '%s': %s", imgname, err)
		os.Exit(1)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.IntVar(&Size, "s", 250, "Image size")
	flag.IntVar(&Samples, "k", 1, "Number of samples per pixel")
	flag.Parse()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		render(scanner.Text())
	}
}
