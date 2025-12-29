package dns

import (
	"encoding/binary"
	"fmt"
)

// DNSAnswer represents a DNS answer section
type DNSAnswer struct {
	Name     []byte
	Type     uint16
	Class    uint16
	TTL      uint32
	RDLength uint16
	RData    []byte
}

// Parse extracts answer section from DNS message
// Returns the number of bytes consumed
func (a *DNSAnswer) Parse(data []byte, offset int) (int, error) {
	// Parse name (can be compressed)
	name, bytesConsumed, err := DecodeName(data, offset)
	if err != nil {
		return 0, fmt.Errorf("failed to parse answer name: %v", err)
	}

	currentOffset := offset + bytesConsumed

	// Need 10 more bytes: Type(2) + Class(2) + TTL(4) + RDLength(2)
	if currentOffset+10 > len(data) {
		return 0, fmt.Errorf("insufficient data for answer fields")
	}

	a.Name = name
	a.Type = binary.BigEndian.Uint16(data[currentOffset : currentOffset+2])
	currentOffset += 2
	a.Class = binary.BigEndian.Uint16(data[currentOffset : currentOffset+2])
	currentOffset += 2
	a.TTL = binary.BigEndian.Uint32(data[currentOffset : currentOffset+4])
	currentOffset += 4
	a.RDLength = binary.BigEndian.Uint16(data[currentOffset : currentOffset+2])
	currentOffset += 2

	// Read RData
	if currentOffset+int(a.RDLength) > len(data) {
		return 0, fmt.Errorf("insufficient data for RData")
	}

	a.RData = make([]byte, a.RDLength)
	copy(a.RData, data[currentOffset:currentOffset+int(a.RDLength)])
	currentOffset += int(a.RDLength)

	return currentOffset, nil
}

// Encode converts a DNS Answer to bytes
func (a *DNSAnswer) Encode() []byte {
	// total length: Name + Type(2) + Class(2) + TTL(4) + RDLength(2) + RData
	buf := make([]byte, len(a.Name)+10+len(a.RData))

	offset := 0
	copy(buf[offset:], a.Name)
	offset += len(a.Name)

	binary.BigEndian.PutUint16(buf[offset:], a.Type)
	offset += 2
	binary.BigEndian.PutUint16(buf[offset:], a.Class)
	offset += 2
	binary.BigEndian.PutUint32(buf[offset:], a.TTL)
	offset += 4
	binary.BigEndian.PutUint16(buf[offset:], a.RDLength)
	offset += 2
	copy(buf[offset:], a.RData)

	return buf
}
