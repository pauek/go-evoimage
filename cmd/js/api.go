package main

import (
	eimg "go-evoimage"
	"math/rand"
	"time"

	"github.com/gopherjs/gopherjs/js"
)

func printCircuit(C eimg.Circuit) string {
	return C.String()
}

func main() {
	js.Global.Set("evoimage", map[string]interface{}{
		"Read":          eimg.Read,
		"RandomCircuit": eimg.RandomCircuit,
	})
	rand.Seed(time.Now().UnixNano())
}
