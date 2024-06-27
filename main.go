package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("====== DECODING ======")
	instructionsPath := os.Args[1]
	instructions := Decode(instructionsPath)

	fmt.Println("")
	fmt.Println("====== EXECUTING ======")
	Execute(instructions)
}
