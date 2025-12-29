package dns

import (
	"encoding/binary"
	"fmt"
)

// DNSHeader represents the DNS header section
type DNSHeader struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

// Parse extracts DNS header fields from the first 12 bytes
func (h *DNSHeader) Parse(data []byte) error {
	if len(data) < 12 {
		return fmt.Errorf("data too short for header")
	}

	h.ID = binary.BigEndian.Uint16(data[0:2])
	h.Flags = binary.BigEndian.Uint16(data[2:4])
	h.QDCount = binary.BigEndian.Uint16(data[4:6])
	h.ANCount = binary.BigEndian.Uint16(data[6:8])
	h.NSCount = binary.BigEndian.Uint16(data[8:10])
	h.ARCount = binary.BigEndian.Uint16(data[10:12])
	return nil
}

// BuildResponse creates a response header based on the request
func (h *DNSHeader) BuildResponse() DNSHeader {
	// Extract OPCODE from request flags (bits 11-14)
	opcode := (h.Flags >> 11) & 0x0F

	// Determine RCODE based on OPCODE
	var rcode uint16
	if opcode != 0 {
		rcode = 4 // Not implemented for non-standard queries
	} else {
		rcode = 0 // No error for standard queries
	}

	// Build response flags:
	// QR=1 (response), OPCODE from request, AA=0, TC=0, RD from request
	// RA=0, Z=0, RCODE as determined above
	var flags uint16
	flags |= 1 << 15            // QR = 1 (response)
	flags |= opcode << 11       // OPCODE from request
	flags |= (h.Flags & 0x0100) // Copy RD (recursion desired)
	flags |= rcode              // Set RCODE (bits 0-3)

	return DNSHeader{
		ID:      h.ID,
		Flags:   flags,
		QDCount: h.QDCount,
		ANCount: 0, // No answers yet
		NSCount: 0,
		ARCount: 0,
	}
}

// Encode converts a DNS header to bytes (12 bytes)
func (h *DNSHeader) Encode() []byte {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint16(buf[0:2], h.ID)
	binary.BigEndian.PutUint16(buf[2:4], h.Flags)
	binary.BigEndian.PutUint16(buf[4:6], h.QDCount)
	binary.BigEndian.PutUint16(buf[6:8], h.ANCount)
	binary.BigEndian.PutUint16(buf[8:10], h.NSCount)
	binary.BigEndian.PutUint16(buf[10:12], h.ARCount)
	return buf
}
