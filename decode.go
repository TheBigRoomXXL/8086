package main

import (
	"fmt"
	"io"
	"strings"
)

type Instruction struct {
	operator     string
	operandLeft  string
	operandRight string
	w            byte
}

func (i *Instruction) String() string {
	if i.operandRight == "" {
		return fmt.Sprintf("%s %s",
			i.operator,
			i.operandLeft,
		)
	}

	return fmt.Sprintf("%s %s, %s",
		i.operator,
		i.operandLeft,
		i.operandRight,
	)
}

// Decode the next instruction in the instruction bus
func Decode(bus io.Reader) (Instruction, error) {

	buffer := make([]byte, 1)

	_, err := bus.Read(buffer)
	if err != nil {
		if err == io.EOF {
			return Instruction{}, err
		}
		panic(err)
	}

	// Each instruction is at least 1 byte long.
	// This byte contain the operation code.
	// Each operation code require different parsing rule.
	// Each operator can have multiple operation code.
	// Some operation code can represent multiple operator.
	opcode := buffer[0] >> 2 // Operation code
	decoder := decoders[opcode]
	if decoder == nil {
		panic(
			fmt.Sprintf(
				"Decoder for opcode %06b not implemented.", opcode,
			),
		)
	}

	return decoder(buffer, bus), nil
}

// ====================
// ===== DECODERS =====
// ====================

// It's possible to write a generic parser for 8086 but it's complicated and
// require extensive knowledge of all the small exceptions contained in the
// 8086 encoding. So i will not try do do that and instead group instructions
// by there type of memory / register / immediate access as it is the main
// determinator of how an instruction will be parsed.

func decodeRegMemToFromReg(buffer []byte, bus io.Reader) Instruction {

	// Parse First byte
	opcode := buffer[0] >> 2 // Operation code
	operator, ok := operators[opcode]
	if !ok {
		panic(fmt.Sprintf("operator for opcode %06b not found", opcode))
	}

	d := buffer[0] >> 1 & 1 // direction to/from register
	w := buffer[0] & 1      // word/byte operator

	// Parse second byte
	checkRead(bus.Read(buffer))

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
		operand2 = getMemoryCalculation(mod, rm, bus)
	}

	operand2 = strings.ReplaceAll(operand2, " + 0", "")

	// Handle direction swap (write instead of read)
	if d == 0 {
		return Instruction{
			operator,
			operand2,
			operand1,
			w,
		}
	}

	return Instruction{
		operator,
		operand1,
		operand2,
		w,
	}
}

func decodeImediateToRegister(buffer []byte, bus io.Reader) Instruction {
	// Parse first byte
	opcode := buffer[0] >> 2 // Operation code
	operator, ok := operators[opcode]
	if !ok {
		panic(fmt.Sprintf("operator for opcode %06b not found", opcode))
	}

	w := buffer[0] & 0b00001000 >> 3
	reg := buffer[0] & 0b00000111

	regKey := reg<<1 | w
	operand1, ok := registers[regKey]
	if !ok {
		panic(fmt.Sprintf("register for %06b not found", regKey))
	}

	// Parse the immediate
	operand2 := ""
	if w == 0 {
		data := getData8(bus)
		operand2 = fmt.Sprintf("%d", data)
	} else {
		data := getData16(bus)
		operand2 = fmt.Sprintf("%d", data)
	}

	return Instruction{
		operator,
		operand1,
		operand2,
		w,
	}
}

func decodeImediateToregisterMemory(buffer []byte, bus io.Reader) Instruction {
	// Parse first byte
	s := buffer[0] >> 1 & 1
	w := buffer[0] & 1

	// Parse second byte
	checkRead(bus.Read(buffer))

	mod := buffer[0] >> 6 // Register / memory mode
	rm := buffer[0] & 7   // Register operand/extension to use in EA calculation

	// Result
	opcodeHint := buffer[0] >> 3 & 0b111
	operator := operatorsArithmetic[opcodeHint]

	operand1 := ""
	if mod == 0b11 {
		regkey := rm<<1 | w
		op, ok := registers[regkey]
		if !ok {
			panic(fmt.Sprintf("register for %06b not found", regkey))
		}
		operand1 = op

	} else {
		operand1 = getMemoryCalculation(mod, rm, bus)
	}

	operand2 := ""
	if s == 0 && w == 0 {
		operand2 = fmt.Sprintf("%d", uint8(getData8(bus)))
	} else if s == 1 && w == 0 {
		operand2 = fmt.Sprintf("%d", getData8(bus))
	} else if s == 0 && w == 1 {
		operand2 = fmt.Sprintf("%d", uint16(getData16(bus)))
	} else if s == 1 && w == 1 {
		operand2 = fmt.Sprintf("%d", getData8(bus))
	} else {
		panic("should not happen")
	}

	return Instruction{
		operator,
		operand1,
		operand2,
		w,
	}
}

func decodeImediateToAccumulator(buffer []byte, bus io.Reader) Instruction {
	opcode := buffer[0] >> 2 // Operation code
	operator, ok := operators[opcode]
	if !ok {
		panic(fmt.Sprintf("operator for opcode %06b not found", opcode))
	}
	w := buffer[0] & 1

	// Accumulator is just a fancy name for the register A
	operand1 := ""
	if w == 0 {
		operand1 = "al"

	} else {
		operand1 = "ax"
	}

	// Parsing second (and potentially third) byte
	operand2 := ""
	if w == 0 {
		data := getData8(bus)
		operand2 = fmt.Sprintf("%d", data)
	} else {
		data := getData16(bus)
		operand2 = fmt.Sprintf("%d", data)
	}

	return Instruction{
		operator,
		operand1,
		operand2,
		w,
	}
}

func decodeCondJumpAndLoop(buffer []byte, bus io.Reader) Instruction {
	operatorHint := buffer[0] & 0b11111
	operator, ok := operatorsJumps[operatorHint]
	if !ok {
		panic("jump operator for %05b not found")
	}

	location := getData8(bus)

	return Instruction{
		operator,
		fmt.Sprintf("%d", location),
		"",
		0,
	}
}

// =================
// ===== UTILS =====
// =================

func checkRead(_ int, err error) {
	if err != nil {
		panic(err)
	}
}

func getData8(bus io.Reader) int8 {
	buffer := make([]byte, 1)
	checkRead(bus.Read(buffer))
	return int8(buffer[0])
}

func getData16(bus io.Reader) int16 {
	buffer := make([]byte, 2)
	checkRead(bus.Read(buffer))
	return int16(buffer[1])<<8 | int16(buffer[0])
}

func getMemoryCalculation(mod byte, rm byte, bus io.Reader) string {
	switch mod {
	case 0b00: // Memory Mode, no displacement
		addrKey := mod<<3 | rm
		return addressCalculations[addrKey]
	case 0b01: // Memory Mode, 8-bit displacement
		addrKey := mod<<3 | rm
		addr := addressCalculations[addrKey]

		d8 := fmt.Sprintf("%d", getData8(bus))
		return strings.Replace(addr, "D8", d8, 1)

	case 0b10: //Memory Mode, 16-bit displacement
		addrKey := mod<<3 | rm
		addr := addressCalculations[addrKey]

		d16 := fmt.Sprintf("%d", getData16(bus))
		return strings.Replace(addr, "D16", d16, 1)

	case 0b11: // Register Mode, no displacement
		panic("No memory calculation when MOD == 0b11")
	}
	panic("No match for MOD")
}

// ====================
// ====== TABLES ======
// ====================

// A choose to use a table driven approche as using tables to parse
// instructions is simple, efficient and extendable.

// The following tables could be optimise to be more generic if they encoded
// the protocol in a smarter way. But 8086 encoding is a mess and I will not
// maintain this codebase so i am just keeping things simple, not doing
// anything smart.

// Reference table 4-12 8086 Instruction Encoding
var decoders = map[byte]func([]byte, io.Reader) Instruction{
	0b100010: decodeRegMemToFromReg,          // MOV
	0b101100: decodeImediateToRegister,       // MOV
	0b101101: decodeImediateToRegister,       // MOV
	0b101110: decodeImediateToRegister,       // MOV
	0b101111: decodeImediateToRegister,       // MOV
	0b100000: decodeImediateToregisterMemory, // ADD SUB CMP
	0b000000: decodeRegMemToFromReg,          // ADD
	0b000001: decodeImediateToAccumulator,    // ADD
	0b001010: decodeRegMemToFromReg,          // SUB
	0b001011: decodeImediateToAccumulator,    // SUB
	0b001110: decodeRegMemToFromReg,          // CMP
	0b001111: decodeImediateToAccumulator,    // CMP
	0b011100: decodeCondJumpAndLoop,          // CONDITIONAL JUMPS
	0b011101: decodeCondJumpAndLoop,          // CONDITIONAL JUMPS
	0b011110: decodeCondJumpAndLoop,          // CONDITIONAL JUMPS
	0b011111: decodeCondJumpAndLoop,          // CONDITIONAL JUMPS
	0b111000: decodeCondJumpAndLoop,          // LOOP
}

var operators = map[byte]string{
	0b100010: "mov",
	0b101100: "mov",
	0b101101: "mov",
	0b101110: "mov",
	0b101111: "mov",
	0b000000: "add",
	0b000001: "add",
	0b001010: "sub",
	0b001011: "sub",
	0b001110: "cmp",
	0b001111: "cmp",
	0b011101: "jnz",
}

// The key is the last 5 bits of the first byte
var operatorsJumps = map[byte]string{
	0b10100: "je",
	0b11100: "jl",
	0b11110: "jle",
	0b10010: "jb",
	0b10110: "jbe",
	0b11010: "jp",
	0b10000: "jo",
	0b11000: "js",
	0b10101: "jnz",
	0b11101: "jge",
	0b11111: "jg",
	0b10011: "jnb",
	0b10111: "ja",
	0b11011: "jpo",
	0b10001: "jno",
	0b11001: "jns",
	0b00010: "loop",
	0b00001: "loopz",
	0b00000: "loopnz",
	0b00011: "jcxz",
}

var operatorsArithmetic = map[byte]string{
	0b000: "add",
	0b101: "sub",
	0b111: "cmp",
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
	0b1011: "dp",
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
