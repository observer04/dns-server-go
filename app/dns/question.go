package dns

import (
	"encoding/binary"
	"fmt"
)

// Question represents a DNS question section
type Question struct {
	QName  []byte
	QType  uint16
	QClass uint16
}

// Parse extracts question section from DNS message
// Returns the number of bytes consumed
func (q *Question) Parse(data []byte, offset int) (int, error) {
	name, bytesConsumed, err := decodeName(data, offset)
	if err != nil {
		return 0, err
	}

	// Update offset to after the QNAME
	currentOffset := offset + bytesConsumed

	if currentOffset+4 > len(data) {
		return 0, fmt.Errorf("insufficient data for QTYPE/QCLASS")
	}

	q.QName = name
	q.QType = binary.BigEndian.Uint16(data[currentOffset : currentOffset+2])
	q.QClass = binary.BigEndian.Uint16(data[currentOffset+2 : currentOffset+4])

	return currentOffset + 4, nil
}

func decodeName(data []byte, offset int) ([]byte, int, error) {
	var name []byte
	bytesConsumed := 0
	currentOffset := offset
	jumped := false

	// Safety limit to prevent infinite loops
	maxLoops := 1000
	loops := 0

	for {
		loops++
		if loops > maxLoops {
			return nil, 0, fmt.Errorf("too many jumps or labels")
		}

		if currentOffset >= len(data) {
			return nil, 0, fmt.Errorf("offset out of bounds")
		}

		b := data[currentOffset]

		// Check for pointer (11xxxxxx)
		if b&0xC0 == 0xC0 {
			if currentOffset+1 >= len(data) {
				return nil, 0, fmt.Errorf("pointer incomplete")
			}

			// Pointer consumes 2 bytes at the original position
			if !jumped {
				bytesConsumed += 2
			}

			// Read pointer value (14 bits)
			ptr := binary.BigEndian.Uint16(data[currentOffset : currentOffset+2])
			newOffset := int(ptr & 0x3FFF)

			currentOffset = newOffset
			jumped = true
			continue
		}

		// Regular label
		if !jumped {
			bytesConsumed += 1
		}
		currentOffset++

		if b == 0 {
			name = append(name, 0)
			break
		}

		labelLen := int(b)
		if currentOffset+labelLen > len(data) {
			return nil, 0, fmt.Errorf("label length out of bounds")
		}

		name = append(name, b)
		name = append(name, data[currentOffset:currentOffset+labelLen]...)

		currentOffset += labelLen
		if !jumped {
			bytesConsumed += labelLen
		}
	}

	return name, bytesConsumed, nil
}

// Encode converts a Question to bytes
func (q *Question) Encode() []byte {
	buf := make([]byte, len(q.QName)+4)
	copy(buf, q.QName)
	binary.BigEndian.PutUint16(buf[len(q.QName):len(q.QName)+2], q.QType)
	binary.BigEndian.PutUint16(buf[len(q.QName)+2:len(q.QName)+4], q.QClass)
	return buf
}
