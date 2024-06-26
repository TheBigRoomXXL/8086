package main

import (
	"encoding/binary"
	"fmt"
	"strconv"
)

type VM struct {
	registers [16]byte          // 8 16bits register
	memory    [1024 * 1024]byte // 1Mb memory
}

func (vm VM) PrintRegistersBinary() {
	r := vm.registers

	fmt.Printf("     ┌─────────────────────┐\n")
	fmt.Printf("     │      REGISTERS      │\n")
	fmt.Printf("┌────┼──────────┬──────────│\n")
	fmt.Printf("│ ax │ %08b │ %08b │\n", r[0], r[1])
	fmt.Printf("│ bx │ %08b │ %08b │\n", r[2], r[3])
	fmt.Printf("│ cx │ %08b │ %08b │\n", r[4], r[5])
	fmt.Printf("│ dx │ %08b │ %08b │\n", r[6], r[7])
	fmt.Printf("├────┼──────────┴──────────┤\n")
	fmt.Printf("│ sp │ %08b   %08b │\n", r[8], r[9])
	fmt.Printf("│ dp │ %08b   %08b │\n", r[10], r[11])
	fmt.Printf("│ si │ %08b   %08b │\n", r[12], r[13])
	fmt.Printf("│ di │ %08b   %08b │\n", r[14], r[15])
	fmt.Printf("└────┴─────────────────────┘\n")
}

func (vm VM) PrintRegistersHex() {
	r := vm.registers

	fmt.Printf("     ┌─────────────┐\n")
	fmt.Printf("     │  REGISTERS  │\n")
	fmt.Printf("┌────┼──────┬──────│\n")
	fmt.Printf("│ ax │ 0x%02x │ 0x%02x │\n", r[0], r[1])
	fmt.Printf("│ bx │ 0x%02x │ 0x%02x │\n", r[2], r[3])
	fmt.Printf("│ cx │ 0x%02x │ 0x%02x │\n", r[4], r[5])
	fmt.Printf("│ dx │ 0x%02x │ 0x%02x │\n", r[6], r[7])
	fmt.Printf("├────┼──────┴──────┤\n")
	fmt.Printf("│ sp │ 0x%02x   0x%02x │\n", r[8], r[9])
	fmt.Printf("│ dp │ 0x%02x   0x%02x │\n", r[10], r[11])
	fmt.Printf("│ si │ 0x%02x   0x%02x │\n", r[12], r[13])
	fmt.Printf("│ di │ 0x%02x   0x%02x │\n", r[14], r[15])
	fmt.Printf("└────┴─────────────┘\n")
}

func Execute(instructions []Instruction) {
	vm := VM{}
	for _, i := range instructions {
		switch i.operator {
		case "mov":
			mov(&vm, i)
		default:
			panic(fmt.Sprintf("Operator %s not implemented", i.operator))
		}
	}

	vm.PrintRegistersHex()

}

func mov(vm *VM, i Instruction) {
	offset := registersOffsets[i.operandLeft]

	value, err := strconv.ParseInt(i.operandRight, 10, 16)
	if err != nil {
		panic(err)
	}

	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(value))
	copy(vm.registers[offset:], b[0:01+i.w])

}

// The following table represent the beginning of each register in our array
var registersOffsets = map[string]int8{
	"ax": 0,
	"ah": 0,
	"al": 1,
	"bx": 2,
	"bh": 2,
	"bl": 3,
	"cx": 4,
	"ch": 4,
	"cl": 5,
	"dx": 6,
	"dh": 6,
	"dl": 7,
	"sp": 8,
	"dp": 10,
	"si": 12,
	"di": 14,
}
