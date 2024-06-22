package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func checkRead(_ int, err error) {
	if err != nil {
		panic(err)
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
		decoder := decoders[opcode]
		if decoder == nil {
			panic(
				fmt.Sprintf(
					"Decoder for opcode %06b not implemented yet", opcode,
				),
			)
		}

		instruction := decoder(buffer, file)
		fmt.Println(instruction)
	}
}

// mov Register/memory to/from register
func decode100010(buffer []byte, file *os.File) string {
	const operator = "mov"
	d := buffer[0] & 2 >> 1 // direction to/from register
	w := buffer[0] & 1      // word/byte operator

	// Parse second byte
	checkRead(file.Read(buffer))

	mod := buffer[0] >> 6     // Register / memory mode
	reg := buffer[0] >> 3 & 7 // Register operand/extension of opcode
	rm := buffer[0] & 7       // Register operand/extension to use in EA calculation

	// Result
	operand1 := ""
	regkey := reg<<1 | w
	operand1 = registers[regkey]

	operand2 := ""
	if mod == 0b11 {
		regKey2 := rm<<1 | w
		operand2 = registers[regKey2]
	} else {
		operand2 = getMemoryCalculation(mod, rm, file)
	}

	operand2 = strings.ReplaceAll(operand2, " + 0", "")

	// Handle direction swap (write instead of read)
	if d == 0 {
		return fmt.Sprintf("%s %s, %s",
			operator,
			operand2,
			operand1,
		)
	}

	return fmt.Sprintf("%s %s, %s",
		operator,
		operand1,
		operand2,
	)
}

// mov immediate to register
func decode1011(buffer []byte, file *os.File) string {
	const operator = "mov"

	// Parse first byte
	w := buffer[0] & 0b00001000 >> 3
	reg := buffer[0] & 0b00000111

	regKey := reg<<1 | w
	operand1 := registers[regKey]

	// Parse the immediate
	operand2 := ""
	if w == 0 {
		data := getData8(file)
		operand2 = fmt.Sprintf("%d", data)
	} else {
		data := getData16(file)
		operand2 = fmt.Sprintf("%d", data)
	}

	return fmt.Sprintf("%s %s, %s",
		operator,
		operand1,
		operand2,
	)
}

func getData8(file *os.File) int8 {
	buffer := make([]byte, 1)
	checkRead(file.Read(buffer))
	return int8(buffer[0])
}

func getData16(file *os.File) int16 {
	buffer := make([]byte, 2)
	checkRead(file.Read(buffer))
	return int16(buffer[1])<<8 | int16(buffer[0])
}

func getMemoryCalculation(mod byte, rm byte, file *os.File) string {
	switch mod {
	case 0b00: // Memory Mode, no displacement
		addrKey := mod<<3 | rm
		return addressCalculations[addrKey]
	case 0b01: // Memory Mode, 8-bit displacement
		addrKey := mod<<3 | rm
		addr := addressCalculations[addrKey]

		d8 := fmt.Sprintf("%d", getData8(file))
		return strings.Replace(addr, "D8", d8, 1)

	case 0b10: //Memory Mode, 16-bit displacement
		addrKey := mod<<3 | rm
		addr := addressCalculations[addrKey]

		d16 := fmt.Sprintf("%d", getData16(file))
		return strings.Replace(addr, "D16", d16, 1)

	case 0b11: // Register Mode, no displacement
		panic("No memory calculation when MOD == 0b11")
	}
	panic("No match for MOD")
}

// Reference table 4-12 8086 Instruction Encoding
var decoders = map[uint8]func([]byte, *os.File) string{
	0b100010: decode100010, // mov register/memory to/from register
	// 0b110001: // mov immediate to register/memory
	0b101100: decode1011, // mov immediate to register
	0b101101: decode1011, // mov immediate to register
	0b101110: decode1011, // mov immediate to register
	0b101111: decode1011, // mov immediate to register
	// 0b101000: // mov memory to accumulator / Accumulator to memory
	// 0b100011: // mov megister/memory to segment register / Segment register to register/memory
}

// Reference Table 4-9 Register Encoding
// Fist 3 bits come from REG (or RM if MOD=0b11) and last one from W
var registers = map[byte]string{
	0b0000: "al",
	0b0010: "cl",
	0b0100: "dl",
	0b0110: "bl",
	0b1000: "ah",
	0b1010: "ch",
	0b1100: "dh",
	0b1110: "bh",
	0b0001: "ax",
	0b0011: "cx",
	0b0101: "dx",
	0b0111: "bx",
	0b1001: "sp",
	0b1011: "bp",
	0b1101: "si",
	0b1111: "di",
}

// Reference table 4-10 Register/Memory Field Encoding
// First 2 bits are MOD, tree next are RM
// MOD cannot be 11 as it mean a register encoding, not memory
var addressCalculations = map[byte]string{
	0b00000: "[bx + si]",
	0b00001: "[bx + di]",
	0b00010: "[bp + si]",
	0b00011: "[bp + di]",
	0b00100: "[si]",
	0b00101: "[di]",
	0b00110: "DIRECT ADDRESS",
	0b00111: "[bx",
	0b01000: "[bx + si + D8]",
	0b01001: "[bx + di + D8]",
	0b01010: "[bp + si + D8]",
	0b01011: "[bp + di + D8]",
	0b01100: "[si + D8]",
	0b01101: "[di + D8]",
	0b01110: "[bp + D8]",
	0b01111: "[bx + D8]",
	0b10000: "[bx + si + D16]",
	0b10001: "[bx + di + D16]",
	0b10010: "[bp + si + D16]",
	0b10011: "[bp + di + D16]",
	0b10100: "[si + D16]",
	0b10101: "[di + D16]",
	0b10110: "[bp + D16]",
	0b10111: "[bx + D16]",
}
