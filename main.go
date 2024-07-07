package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] <path-to-instructions>\n", os.Args[0])
		flag.PrintDefaults()
	}

	decodeFlag := flag.Bool(
		"decode",
		false,
		"Only decode the instructions, do not execute them.",
	)
	binaryFlag := flag.Bool(
		"binary",
		false,
		"Print the final state of register in binary format",
	)
	dumpFlag := flag.Bool(
		"dump",
		false,
		"Dump memory into a `memory.data` file at the end of the program",
	)
	flag.Parse()

	// Open file with assembly insructions to decode
	filePath := flag.Arg(0)

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	Execute(file, *decodeFlag, !*binaryFlag, *dumpFlag)
}
