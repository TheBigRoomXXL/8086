package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	filePath := os.Args[1]

	// Prpare the raw assembly insruction to decode
	assembly, err := os.Open(filePath)
	if err != nil {
		panic("fail to open file")
	}
	defer assembly.Close()

	for {
		if decodeInstruction(assembly) {
			break
		}
	}

}

func decodeInstruction(assembly *os.File) bool {
	// b1 and b2 could be the same buffer but it's easier for debug

	// Parse byte 1
	b1 := make([]byte, 1)
	if checkRead(assembly.Read(b1)) != nil {
		return true
	}

	opcode := b1[0] >> 2 // Operation code
	d := b1[0] & 2       // Direction to/from register
	w := b1[0] & 1       // Word/byte operator

	// Parse byte 2
	b2 := make([]byte, 1)
	if checkRead(assembly.Read(b2)) != nil {
		return true
	}

	mod := b2[0] >> 6     // Register / memory mode
	reg := b2[0] >> 3 & 7 // Register operand/extension of opcode
	rm := b2[0] & 7       // Register operand/extension to use in EA calculation

	// // Debug
	// fmt.Printf("%b %b \n", b1[0], b2[0])

	// fmt.Printf("opcode %06b - %s \n", opcode, opcodeToInstruct[opcode])
	// fmt.Printf("d      %01b \n", d)
	// fmt.Printf("w      %01b \n", w)
	// fmt.Printf("mod    %02b - %s \n", mod, modeToLabel[int8(mod)])
	// fmt.Printf("reg    %03b \n", reg)
	// fmt.Printf("rm     %03b \n", rm)

	// Result
	operand1 := byte(0)
	operand2 := byte(0)

	if mod != 0b11 {
		panic("not implemented yet")
	}

	if d == 0 {
		operand1 = rm<<1 | w
		operand2 = reg<<1 | w
	} else {
		operand1 = reg<<1 | w
		operand2 = rm<<1 | w
	}

	fmt.Printf("%s %s, %s\n",
		opcodeToInstruct[opcode],
		registers[int8(operand1)],
		registers[int8(operand2)],
	)

	return false
}

func checkRead(n int, err error) error {
	if err != nil {
		if err == io.EOF {
			return err
		}
		panic("fuck")
	}
	if n == 0 {
		panic("nope")
	}
	return nil
}

var opcodeToInstruct = map[uint8]string{
	0b100010: "MOV",
}

// Reference Table 4-8 MOD(Mode) Field Encoding
var modeToLabel = map[int8]string{
	0b00: "Memory Mode, no displacement",
	0b01: "Memory Mode, 8-bit displacement",
	0b10: "Memory Mode, 16-bit displacement",
	0b11: "Register Mode, no displacement",
}

// Reference Table 4-9 REG(Register) Encoding
// This table take 4 bits. The 3 from REG fist then the one from W
var registers = map[int8]string{
	0b0000: "AL",
	0b0010: "CL",
	0b0100: "DL",
	0b0110: "BL",
	0b1000: "AH",
	0b1010: "CH",
	0b1100: "DH",
	0b1110: "BH",
	0b0001: "AX",
	0b0011: "CX",
	0b0101: "DX",
	0b0111: "BX",
	0b1001: "SP",
	0b1011: "BP",
	0b1101: "SI",
	0b1111: "DI",
}
