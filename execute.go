package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
)

func Execute(bus io.ReadSeeker) {
	vm := VM{}
	for {
		i, err := Decode(bus)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		fmt.Printf("%s: ", &i)

		switch i.operator {
		case "mov":
			vm.mov(i)
		case "add":
			vm.add(i)
		case "sub":
			vm.sub(i)
		case "cmp":
			vm.cmp(i)
		default:
			panic(fmt.Sprintf("Operator %s not implemented", i.operator))
		}
		fmt.Print("\n")
	}

	vm.PrintRegistersHex()

}

// ========================
// ===== INSTRUCTIONS =====
// ========================

func (vm *VM) mov(i Instruction) {
	size := int8(1 + i.w)
	value := vm.getOperandAsBytes(i.operandRight, size)
	vm.setRegister(i.operandLeft, value)
}

func (vm *VM) add(i Instruction) {
	size := int8(1 + i.w)

	valueA := vm.getOperandAsInt(i.operandLeft, size)
	valueB := vm.getOperandAsInt(i.operandRight, size)

	valueInt := valueA + valueB
	valueBytes := make([]byte, size)
	binary.LittleEndian.PutUint16(valueBytes, valueInt)

	vm.setRegister(i.operandLeft, valueBytes)

	vm.setZeroFlag(valueInt == 0)
	vm.setSignFlag(valueBytes[size-1]>>7 == 1)
}

func (vm *VM) sub(i Instruction) {
	size := int8(1 + i.w)

	valueA := vm.getOperandAsInt(i.operandLeft, size)
	valueB := vm.getOperandAsInt(i.operandRight, size)

	valueInt := valueA - valueB
	valueBytes := make([]byte, size)
	binary.LittleEndian.PutUint16(valueBytes, valueInt)

	vm.setRegister(i.operandLeft, valueBytes)

	vm.setZeroFlag(valueInt == 0)
	vm.setSignFlag(valueBytes[size-1]>>7 == 1)
}

func (vm *VM) cmp(i Instruction) {
	size := int8(1 + i.w)

	valueA := vm.getOperandAsInt(i.operandLeft, size)
	valueB := vm.getOperandAsInt(i.operandRight, size)

	fmt.Print("writing nothing to storage ")

	valueInt := valueA - valueB
	valueBytes := make([]byte, size)
	binary.LittleEndian.PutUint16(valueBytes, valueInt)

	vm.setZeroFlag(valueInt == 0)
	vm.setSignFlag(valueBytes[size-1]>>7 == 1)
}

// =================
// ===== UTILS =====
// =================

// Return the imediate value or lookup the register.
// Memory acces not implemented.
func (vm *VM) getOperandAsBytes(operand string, size int8) []byte {
	value := make([]byte, size)
	tmp, err := strconv.ParseInt(operand, 10, 16)
	if err == nil {
		// Then operandRight is an imediate
		binary.LittleEndian.PutUint16(value, uint16(tmp))
	} else {
		// Then operandRight is a register / memory
		offset := registersOffsets[operand]
		value = vm.storage[offset : offset+size]
	}
	return value
}

// Save as getOperandAsBytes but converted to int with littleEndian format.
func (vm *VM) getOperandAsInt(operand string, size int8) uint16 {
	value := uint16(0)
	tmp, err := strconv.ParseInt(operand, 10, 16)
	if err == nil {
		// Then operandRight is an imediate
		value = uint16(tmp)

	} else {
		// Then operandRight is a register / memory
		offset := registersOffsets[operand]
		value = binary.LittleEndian.Uint16(
			vm.storage[offset : offset+size],
		)
	}
	return value
}

func (vm *VM) setRegister(reg string, value []byte) {
	offset := registersOffsets[reg]
	fmt.Printf("(%s 0x%02x->", reg, vm.storage[offset:offset+2])
	copy(vm.storage[offset:], value)
	fmt.Printf("(0x%02x) ", vm.storage[offset:offset+2])

}

func (vm *VM) setZeroFlag(flag bool) {
	fmt.Printf("(ZF %t) ", flag)
	if flag {
		vm.storage[17] = vm.storage[17] | 0x80
		return
	}
	vm.storage[17] = vm.storage[17] & 0x7F
}

func (vm *VM) setSignFlag(flag bool) {
	fmt.Printf("(SF %t) ", flag)
	if flag {
		vm.storage[16] = vm.storage[16] | 0x01
		return
	}
	vm.storage[16] = vm.storage[16] & 0xFE
}

// I LOVE ASCII TABLES
func (vm *VM) PrintRegistersBinary() {
	r := vm.storage

	fmt.Printf("     ┌─────────────────────┐\n")
	fmt.Printf("     │       STORAGE       │\n")
	fmt.Printf("┌────┼──────────┬──────────┤\n")
	fmt.Printf("│ ax │ %08b │ %08b │\n", r[0], r[1])
	fmt.Printf("│ bx │ %08b │ %08b │\n", r[2], r[3])
	fmt.Printf("│ cx │ %08b │ %08b │\n", r[4], r[5])
	fmt.Printf("│ dx │ %08b │ %08b │\n", r[6], r[7])
	fmt.Printf("├────┼──────────┴──────────┤\n")
	fmt.Printf("│ sp │ %08b   %08b │\n", r[8], r[9])
	fmt.Printf("│ dp │ %08b   %08b │\n", r[10], r[11])
	fmt.Printf("│ si │ %08b   %08b │\n", r[12], r[13])
	fmt.Printf("│ di │ %08b   %08b │\n", r[14], r[15])
	fmt.Printf("├────┼─────────────────────┤\n")
	fmt.Printf("│ fl │ %08b   %08b │\n", r[16], r[17])
	fmt.Printf("└────┴─────────────────────┘\n")
}

// MOOOAAARE ASCII TABLES
func (vm *VM) PrintRegistersHex() {
	r := vm.storage

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
	fmt.Printf("├────┼─────────────┤\n")
	fmt.Printf("│ fl │ 0x%02x   0x%02x │\n", r[16], r[17])
	fmt.Printf("└────┴─────────────┘\n")
}

// ==================
// ===== TABLES =====
// ==================

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
	"fl": 14,
}
