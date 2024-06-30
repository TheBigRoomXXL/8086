package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
)

func Execute(bus io.ReadSeeker) {
	store := Storage{bus, [20]byte{}}
	for {
		i, err := Decode(store.bus)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		fmt.Printf("%s: ", &i)
		store.incrementIP(uint16(i.size))

		execute := executors[i.operator]
		if execute == nil {
			panic(
				fmt.Sprintf("Operation %s in not implemented", i.operandLeft),
			)
		}
		execute(&store, i)

		fmt.Print("\n")
	}

	store.PrintRegistersHex()

}

// ========================
// ===== INSTRUCTIONS =====
// ========================

func mov(store *Storage, i Instruction) {
	size := int8(1 + i.w)
	value := store.getOperandAsBytes(i.operandRight, size)
	store.setRegister(i.operandLeft, value)
}

func add(store *Storage, i Instruction) {
	size := int8(1 + i.w)
	fmt.Println(i)
	fmt.Println(i.w)

	valueA := store.getOperandAsInt(i.operandLeft, size)
	valueB := store.getOperandAsInt(i.operandRight, size)

	valueInt := valueA + valueB
	var valueBytes []byte
	if size == 1 {
		valueBytes = []byte{byte(valueInt)}
	} else {
		valueBytes = make([]byte, size)
		binary.LittleEndian.PutUint16(valueBytes, valueInt)
	}

	store.setRegister(i.operandLeft, valueBytes)

	store.setZeroFlag(valueInt == 0)
	store.setSignFlag(valueBytes[size-1]>>7 == 1)
}

func sub(store *Storage, i Instruction) {
	size := int8(1 + i.w)

	valueA := store.getOperandAsInt(i.operandLeft, size)
	valueB := store.getOperandAsInt(i.operandRight, size)

	valueInt := valueA - valueB
	var valueBytes []byte
	if size == 1 {
		valueBytes = []byte{byte(valueInt)}
	} else {
		valueBytes = make([]byte, size)
		binary.LittleEndian.PutUint16(valueBytes, valueInt)
	}

	store.setRegister(i.operandLeft, valueBytes)

	store.setZeroFlag(valueInt == 0)
	store.setSignFlag(valueBytes[size-1]>>7 == 1)
}

func cmp(store *Storage, i Instruction) {
	size := int8(1 + i.w)

	valueA := store.getOperandAsInt(i.operandLeft, size)
	valueB := store.getOperandAsInt(i.operandRight, size)

	fmt.Print("writing nothing to storage ")

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

	fmt.Printf("Jumping to %d ", offset)

	store.incrementIP(uint16(offset))
	store.bus.Seek(int64(store.getIP()), 1)
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
	bus     io.ReadSeeker // Instruction bus
	storage [20]byte      // 8 * 16bits register + IP register + Flags register
}

// Return the imediate value or lookup the register.
// Memory acces not implemented.
func (store *Storage) getOperandAsBytes(operand string, size int8) []byte {
	value := make([]byte, size)
	tmp, err := strconv.ParseInt(operand, 10, 16)
	if err == nil {
		// Then operandRight is an imediate
		binary.LittleEndian.PutUint16(value, uint16(tmp))
	} else {
		// Then operandRight is a register / memory
		offset := registersOffsets[operand]
		value = store.storage[offset : offset+size]
	}
	return value
}

// Save as getOperandAsBytes but converted to int with littleEndian format.
func (store *Storage) getOperandAsInt(operand string, size int8) uint16 {
	value := uint16(0)
	tmp, err := strconv.ParseInt(operand, 10, 16)
	if err == nil {
		// Then operandRight is an imediate
		value = uint16(tmp)

	} else {
		// Then operandRight is a register / memory
		offset := registersOffsets[operand]
		if size == 1 {
			value = uint16((store.storage[offset]))
		} else {
			value = binary.LittleEndian.Uint16(
				store.storage[offset : offset+size],
			)
		}
	}
	return value
}

func (store *Storage) setRegister(reg string, value []byte) {
	offset := registersOffsets[reg]
	fmt.Printf("(%s 0x%02x->", reg, store.storage[offset:offset+2])
	copy(store.storage[offset:], value)
	fmt.Printf("(0x%02x) ", store.storage[offset:offset+2])

}

func (store *Storage) setZeroFlag(flag bool) {
	fmt.Printf("(ZF %t) ", flag)
	if flag {
		store.storage[19] = store.storage[19] | 0x80
		return
	}
	store.storage[19] = store.storage[19] & 0x7F
}

func (store *Storage) getZeroFlag() bool {
	return store.storage[19]>>7 == 1
}

func (store *Storage) setSignFlag(flag bool) {
	fmt.Printf("(SF %t) ", flag)
	if flag {
		store.storage[18] = store.storage[18] | 0x01
		return
	}
	store.storage[18] = store.storage[18] & 0xFE
}

func (store *Storage) getSignFlag() bool {
	return store.storage[18]&0b1 == 1
}

func (store *Storage) incrementIP(size uint16) {
	current := binary.LittleEndian.Uint16(store.storage[16:18])
	current += uint16(size)

	currentByte := make([]byte, 2)
	binary.LittleEndian.PutUint16(currentByte, current)

	copy(store.storage[16:], currentByte)
	fmt.Printf("(IP 0x%04x) ", currentByte)

}

func (store *Storage) getIP() uint16 {
	return binary.LittleEndian.Uint16(store.storage[16:18])
}

// I LOVE ASCII TABLES
func (store *Storage) PrintRegistersBinary() {
	r := store.storage

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
	fmt.Printf("├────┼─────────────────────┤\n")
	fmt.Printf("│ fl │ %08b   %08b │\n", r[18], r[19])
	fmt.Printf("└────┴─────────────────────┘\n")
}

// MOOOAAARE ASCII TABLES
func (store *Storage) PrintRegistersHex() {
	r := store.storage

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
