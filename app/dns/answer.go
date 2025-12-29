package dns

import (
	"encoding/binary"
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
