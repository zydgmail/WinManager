package guac

import (
	"fmt"
	"strconv"
	"strings"
)

// Instruction represents a Guacamole instruction
type Instruction struct {
	Opcode string
	Args   []string
}

// NewInstruction creates a new instruction
func NewInstruction(opcode string, args ...string) *Instruction {
	return &Instruction{
		Opcode: opcode,
		Args:   args,
	}
}

// Byte converts the instruction to bytes
func (i *Instruction) Byte() []byte {
	return []byte(i.String())
}

// String converts the instruction to string format
func (i *Instruction) String() string {
	var parts []string
	
	// Add opcode
	parts = append(parts, fmt.Sprintf("%d.%s", len(i.Opcode), i.Opcode))
	
	// Add arguments
	for _, arg := range i.Args {
		parts = append(parts, fmt.Sprintf("%d.%s", len(arg), arg))
	}
	
	return strings.Join(parts, ",") + ";"
}

// Parse 解析data到guacd instruction
func Parse(data []byte) (*Instruction, error) {
	elementStart := 0

	// Build list of elements
	elements := make([]string, 0, 1)
	for elementStart < len(data) {
		// Find end of length
		lengthEnd := -1
		for i := elementStart; i < len(data); i++ {
			if data[i] == '.' {
				lengthEnd = i
				break
			}
		}
		
		if lengthEnd == -1 {
			return nil, ErrServer.NewError("ReadSome返回不完整的指令")
		}

		// Parse length
		length, e := strconv.Atoi(string(data[elementStart:lengthEnd]))
		if e != nil {
			return nil, ErrServer.NewError("ReadSome返回错误模式的指令", e.Error())
		}

		// Extract element
		elementStart = lengthEnd + 1
		elementEnd := elementStart + length

		if elementEnd > len(data) {
			return nil, ErrServer.NewError("指令长度超出数据范围")
		}

		element := string(data[elementStart:elementEnd])
		elements = append(elements, element)

		// Move to next element
		elementStart = elementEnd
		
		// Check for separator or terminator
		if elementStart < len(data) {
			if data[elementStart] == ',' {
				elementStart++ // Skip comma
			} else if data[elementStart] == ';' {
				break // End of instruction
			}
		}
	}

	if len(elements) == 0 {
		return nil, ErrServer.NewError("空指令")
	}

	return &Instruction{
		Opcode: elements[0],
		Args:   elements[1:],
	}, nil
}
