package main

import (
	"os"
)

func main() {
	// Open file with assembly insructions to decode
	filePath := os.Args[1]
	file, err := os.Open(filePath)
	if err != nil {
		panic("fail to open file")
	}
	defer file.Close()

	Execute(file)
}
