package main

import "os"

func main() {
	instructionsPath := os.Args[1]
	Decode(instructionsPath)
}
