package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

// DNSMessage represents a complete DNS message
type DNSMessage struct {
	Header    DNSHeader
	Questions []Question
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

// DNSServer handles DNS server operations
type DNSServer struct {
	conn *net.UDPConn
}

// NewDNSServer creates a new DNS server instance
func NewDNSServer(addr string) (*DNSServer, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	return &DNSServer{conn: conn}, nil
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

	// Build response flags:
	// QR=1 (response), OPCODE from request, AA=0, TC=0, RD from request
	// RA=0, Z=0, RCODE=0 (no error)
	var flags uint16
	flags |= 1 << 15            // QR = 1 (response)
	flags |= opcode << 11       // OPCODE from request
	flags |= (h.Flags & 0x0100) // Copy RD (recursion desired)

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
	response := DNSMessage{
		Header:    msg.Header.BuildResponse(),
		Questions: make([]Question, len(msg.Questions)),
	}

	// Copy questions to response
	copy(response.Questions, msg.Questions)

	return response
}

// Encode converts a DNS message to bytes
func (msg *DNSMessage) Encode() []byte {
	buf := msg.Header.Encode()

	// Encode questions
	for _, q := range msg.Questions {
		buf = append(buf, q.Encode()...)
	}

	return buf
}

// HandleQuery processes a DNS query and returns the response
func (s *DNSServer) HandleQuery(data []byte) ([]byte, error) {
	// Parse the request
	var request DNSMessage
	if err := request.Parse(data); err != nil {
		return nil, fmt.Errorf("failed to parse request: %v", err)
	}

	fmt.Printf("Request ID: %d, Flags: 0x%04x, Questions: %d\n",
		request.Header.ID, request.Header.Flags, request.Header.QDCount)

	// Build response
	response := request.BuildResponse()

	// Encode to bytes
	return response.Encode(), nil
}

// Run starts the DNS server
func (s *DNSServer) Run() error {
	defer s.conn.Close()

	buf := make([]byte, 512)

	for {
		size, source, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Printf("Error receiving data: %v\n", err)
			break
		}

		fmt.Printf("Received %d bytes from %s\n", size, source)

		// Handle the query
		response, err := s.HandleQuery(buf[:size])
		if err != nil {
			fmt.Printf("Error handling query: %v\n", err)
			continue
		}

		// Send response
		_, err = s.conn.WriteToUDP(response, source)
		if err != nil {
			fmt.Printf("Failed to send response: %v\n", err)
		}
	}

	return nil
}

func main() {
	fmt.Println("Logs from your program will appear here!")

	// Create and start DNS server
	server, err := NewDNSServer("127.0.0.1:2053")
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}

	fmt.Println("DNS server listening on 127.0.0.1:2053")
	if err := server.Run(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
