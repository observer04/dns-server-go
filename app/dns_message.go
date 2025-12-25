package main

import (
	"encoding/binary"
	"fmt"
)

// DNSMessage represents a complete DNS message
type DNSMessage struct {
	Header    DNSHeader
	Questions []Question
	Answers   []DNSAnswer
}

// DNSHeader represents the DNS header section
type DNSHeader struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

// Question represents a DNS question section
type Question struct {
	QName  []byte
	QType  uint16
	QClass uint16
}

// DNSAnswer represents a DNS answer section
type DNSAnswer struct {
	Name     []byte
	Type     uint16
	Class    uint16
	TTL      uint32
	RDLength uint16
	RData    []byte
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

// Parse extracts question section from DNS message
// Returns the number of bytes consumed
func (q *Question) Parse(data []byte, offset int) (int, error) {
	if offset >= len(data) {
		return 0, fmt.Errorf("offset beyond data length")
	}

	// Parse QNAME (domain name in label format)
	// Each label is: length byte + label bytes
	// Terminated by 0-length byte
	// eg: www.example.com -> 3www7example3com0
	start := offset
	for offset < len(data) && data[offset] != 0 {
		labelLen := int(data[offset])
		offset += 1 + labelLen
		if offset > len(data) {
			return 0, fmt.Errorf("invalid label length")
		}
	}
	offset++ // Skip the terminating 0

	if offset+4 > len(data) {
		return 0, fmt.Errorf("insufficient data for QTYPE/QCLASS")
	}

	q.QName = data[start:offset]
	q.QType = binary.BigEndian.Uint16(data[offset : offset+2])
	q.QClass = binary.BigEndian.Uint16(data[offset+2 : offset+4])

	return offset + 4, nil
}

// Encode converts a Question to bytes
func (q *Question) Encode() []byte {
	buf := make([]byte, len(q.QName)+4)
	copy(buf, q.QName)
	binary.BigEndian.PutUint16(buf[len(q.QName):len(q.QName)+2], q.QType)
	binary.BigEndian.PutUint16(buf[len(q.QName)+2:len(q.QName)+4], q.QClass)
	return buf
}

// Parse extracts a complete DNS message from bytes
func (msg *DNSMessage) Parse(data []byte) error {
	// Parse header
	if err := msg.Header.Parse(data); err != nil {
		return err
	}

	// Parse questions
	offset := 12 // Start after header
	msg.Questions = make([]Question, 0, msg.Header.QDCount)

	for i := uint16(0); i < msg.Header.QDCount; i++ {
		var q Question
		bytesRead, err := q.Parse(data, offset)
		if err != nil {
			return err
		}
		msg.Questions = append(msg.Questions, q)
		offset = bytesRead
	}

	return nil
}

// BuildResponse creates a response message based on the request
func (msg *DNSMessage) BuildResponse() DNSMessage {
	header := msg.Header.BuildResponse()
	header.QDCount = uint16(len(msg.Questions))
	header.ANCount = uint16(len(msg.Questions)) // Set answer count

	response := DNSMessage{
		Header:    header,
		Questions: make([]Question, len(msg.Questions)),
		Answers:   make([]DNSAnswer, 0, len(msg.Questions)),
	}

	// Copy questions and add dummy A records for each
	for i, q := range msg.Questions {
		response.Questions[i] = q
		response.Answers = append(response.Answers, DNSAnswer{
			Name:     q.QName,
			Type:     q.QType, // Use the requested type
			Class:    q.QClass,
			TTL:      60,
			RDLength: 4,
			RData:    []byte{8, 8, 8, 8}, // Dummy IP
		})
	}

	return response
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

// Encode converts a DNS message to bytes
func (msg *DNSMessage) Encode() []byte {
	buf := msg.Header.Encode()

	// Encode questions
	for _, q := range msg.Questions {
		buf = append(buf, q.Encode()...)
	}

	// Encode answers
	for _, a := range msg.Answers {
		buf = append(buf, a.Encode()...)
	}

	return buf
}
