package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
)

func Execute(bus io.ReadSeeker, decodeOnly bool, printHex bool) {
	store := Storage{bus, [20]byte{}, [64 * 1024]byte{}}
	fmt.Print("────────────────────────── EXECUTION ───────────────────────────\n")
	for {
		i, err := Decode(store.bus)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}

		if decodeOnly {
			fmt.Printf("%s\n", &i)
			continue
		}

		fmt.Printf("%- 12s ", &i)
		store.incrementIP(uint16(i.size))

		execute := executors[i.operator]
		if execute == nil {
			panic(
				fmt.Sprintf("Operation %s in not implemented", i.operator),
			)
		}

		execute(&store, i)

		fmt.Print("\n")
	}

	if decodeOnly {
		return
	}

	fmt.Print("\n───────────────────────── FINAL STATE ──────────────────────────\n")
	if printHex {
		store.PrintRegistersHex()
	} else {
		store.PrintRegistersBinary()
	}

}

// ========================
// ===== INSTRUCTIONS =====
// ========================

func mov(store *Storage, i Instruction) {
	size := int8(1 + i.w)
	value := store.read(i.operandRight, size)
	store.write(i.operandLeft, value)
}

func add(store *Storage, i Instruction) {
	size := int8(1 + i.w)

	valueA := store.readAsInt(i.operandLeft, size)
	valueB := store.readAsInt(i.operandRight, size)

	valueInt := valueA + valueB
	var valueBytes []byte
	if size == 1 {
		valueBytes = []byte{byte(valueInt)}
	} else {
		valueBytes = make([]byte, size)
		binary.LittleEndian.PutUint16(valueBytes, valueInt)
	}

	store.write(i.operandLeft, valueBytes)

	store.setZeroFlag(valueInt == 0)
	store.setSignFlag(valueBytes[size-1]>>7 == 1)
}

func sub(store *Storage, i Instruction) {
	size := int8(1 + i.w)

	valueA := store.readAsInt(i.operandLeft, size)
	valueB := store.readAsInt(i.operandRight, size)

	valueInt := valueA - valueB
	var valueBytes []byte
	if size == 1 {
		valueBytes = []byte{byte(valueInt)}
	} else {
		valueBytes = make([]byte, size)
		binary.LittleEndian.PutUint16(valueBytes, valueInt)
	}

	store.write(i.operandLeft, valueBytes)

	store.setZeroFlag(valueInt == 0)
	store.setSignFlag(valueBytes[size-1]>>7 == 1)
}

func cmp(store *Storage, i Instruction) {
	size := int8(1 + i.w)

	valueA := store.readAsInt(i.operandLeft, size)
	valueB := store.readAsInt(i.operandRight, size)

	fmt.Print("writing nothing to internal ")

	valueInt := valueA - valueB
	var valueBytes []byte
	if size == 1 {
		valueBytes = []byte{byte(valueInt)}
	} else {
		valueBytes = make([]byte, size)
		binary.LittleEndian.PutUint16(valueBytes, valueInt)
	}

	store.setZeroFlag(valueInt == 0)
	store.setSignFlag(valueBytes[size-1]>>7 == 1)
}

func jmp(store *Storage, i Instruction) {
	offset, err := strconv.ParseInt(i.operandLeft, 10, 16)
	if err != nil {
		panic(
			fmt.Sprintf("JMP only support immediate value, %s", err),
		)
	}

	fmt.Printf("[jump %d] ", offset)

	store.incrementIP(uint16(offset))
	_, err = store.bus.Seek(offset, 1)
	if err != nil {
		panic(err)
	}

}

// Jump if equal
func je(store *Storage, i Instruction) {
	if store.getZeroFlag() {
		jmp(store, i)
	}
}

// Jump if not equal
func jne(store *Storage, i Instruction) {
	if !store.getZeroFlag() {
		jmp(store, i)
	}
}

// Jump if signed
func js(store *Storage, i Instruction) {
	if store.getSignFlag() {
		jmp(store, i)
	}
}

// Jump if not signed
func jns(store *Storage, i Instruction) {
	if !store.getSignFlag() {
		jmp(store, i)
	}
}

// =================
// ===== UTILS =====
// =================

type Storage struct {
	bus      io.ReadSeeker   // Instruction bus
	internal [20]byte        // 8 * 16bits register + IP register + Flags register
	memory   [64 * 1024]byte // We only have 64Kb of memory because we don't implement segment registers
}

// Return the imediate value or lookup the register.
// Memory acces not implemented.
func (store *Storage) read(location string, size int8) []byte {
	value := make([]byte, size)
	tmp, err := strconv.ParseInt(location, 10, 16)
	if err == nil {
		// Then operandRight is an imediate
		binary.LittleEndian.PutUint16(value, uint16(tmp))
		return value
	}

	// Then operandRight is a register or memory
	offset := registersOffsets[location]
	value = store.internal[offset : offset+size]

	return value
}

// Same as read but converted to int with littleEndian format.
func (store *Storage) readAsInt(location string, size int8) uint16 {
	raw := store.read(location, size)
	return binary.LittleEndian.Uint16(raw)
}

func (store *Storage) write(location string, value []byte) {
	offset, isReg := registersOffsets[location]
	if isReg {
		store.writeToRegister(offset, location, value)
	} else {
		fmt.Print("something something memory")
	}
}

func (store *Storage) writeToRegister(offset int8, reg string, value []byte) {
	fmt.Printf("[%s 0x%02x->", reg, store.internal[offset:offset+2])
	copy(store.internal[offset:], value)
	fmt.Printf("[0x%02x] ", store.internal[offset:offset+2])
}

func (store *Storage) setZeroFlag(flag bool) {
	fmt.Printf("[ZF %t] ", flag)
	if flag {
		store.internal[19] = store.internal[19] | 0x80
		return
	}
	store.internal[19] = store.internal[19] & 0x7F
}

func (store *Storage) getZeroFlag() bool {
	return store.internal[19]>>7 == 1
}

func (store *Storage) setSignFlag(flag bool) {
	fmt.Printf("[SF %t] ", flag)
	if flag {
		store.internal[18] = store.internal[18] | 0x01
		return
	}
	store.internal[18] = store.internal[18] & 0xFE
}

func (store *Storage) getSignFlag() bool {
	return store.internal[18]&0b1 == 1
}

func (store *Storage) incrementIP(size uint16) {
	current := binary.LittleEndian.Uint16(store.internal[16:18])
	current += uint16(size)

	currentByte := make([]byte, 2)
	binary.LittleEndian.PutUint16(currentByte, current)

	copy(store.internal[16:], currentByte)
	fmt.Printf("[IP 0x%04x] ", currentByte)

}

// I LOVE ASCII TABLES
func (store *Storage) PrintRegistersBinary() {
	r := store.internal

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
	fmt.Printf("│ ip │ %08b   %08b │\n", r[16], r[17])
	fmt.Printf("├────┼─────────────────────┤\n")
	fmt.Printf("│ fl │ %08b   %08b │\n", r[18], r[19])
	fmt.Printf("└────┴─────────────────────┘\n")
}

// MOOOAAARE ASCII TABLES
func (store *Storage) PrintRegistersHex() {
	r := store.internal

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
	fmt.Printf("│ ip │ 0x%02x   0x%02x │\n", r[16], r[17])
	fmt.Printf("├────┼─────────────┤\n")
	fmt.Printf("│ fl │ 0x%02x   0x%02x │\n", r[18], r[19])
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

var executors = map[string]func(*Storage, Instruction){
	"mov": mov,
	"add": add,
	"sub": sub,
	"cmp": cmp,
	"jmp": jmp,
	"je":  je,
	"jne": jne,
	"js":  js,
	"jns": jns,
}
