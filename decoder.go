package main

import (
	"fmt"
	"io"
	"os"
)

func checkRead(n int, err error) {
	if err != nil {
		panic(err)
	}
	if n != 1 {
		panic(fmt.Sprintf("wtf, not exactly %d bytes read", n))
	}
}

func main() {
	filePath := os.Args[1]

	// Open file with assembly insructions to decode
	file, err := os.Open(filePath)
	if err != nil {
		panic("fail to open file")
	}
	defer file.Close()

	// Iterate over each instructiob, decoding them one by one
	for {
		// We will parse instructions byte by byte
		buffer := make([]byte, 1)

		_, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}

		// Each instruction is at least 1 byte long.
		// This byte contain the operation code.
		// Each operation code require different parsing rule.
		// Each operator can have multiple operation code.
		opcode := buffer[0] >> 2 // Operation code
		instruction := decoders[opcode](buffer, file)
		fmt.Println(instruction)
	}
}

// Register/memory to/from register
func decode100010(buffer []byte, file *os.File) string {
	const operator = "MOV"
	d := buffer[0] & 2 // direction to/from register
	w := buffer[0] & 1 // word/byte operator

	// Parse second byte
	checkRead(file.Read(buffer))

	mod := buffer[0] >> 6     // Register / memory mode
	reg := buffer[0] >> 3 & 7 // Register operand/extension of opcode
	rm := buffer[0] & 7       // Register operand/extension to use in EA calculation

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

	return fmt.Sprintf("%s %s, %s",
		operator,
		registers[int8(operand1)],
		registers[int8(operand2)],
	)
}

// Reference table 4-12 8086 Instruction Encoding
var decoders = map[uint8]func([]byte, *os.File) string{
	0b100010: decode100010, // MOV register/memory to/from register
	// 0b110001: // MOV immediate to register/memory
	// 0b101100: // MOV immediate to register
	// 0b101101: // MOV immediate to register
	// 0b101110: // MOV immediate to register
	// 0b101111: // MOV immediate to register
	// 0b101000: // MOV memory to accumulator / Accumulator to memory
	// 0b100011: // MOV megister/memory to segment register / Segment register to register/memory
}

// Reference Table 4-8 MOD(Mode) Field Encoding
// var modeToLabel = map[int8]string{
// 	0b00: "Memory Mode, no displacement",
// 	0b01: "Memory Mode, 8-bit displacement",
// 	0b10: "Memory Mode, 16-bit displacement",
// 	0b11: "Register Mode, no displacement",
// }

// Reference Table 4-9 REG(Register) Encoding
// Fist 3 bits come from REG and last one from W
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
