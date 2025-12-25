package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

// dns message struct
type DNSmsg struct {
	Head DNSheader
	Body DNSbody
}

type DNSheader struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

type DNSbody struct {
	QName  []byte
	QType  uint16
	QClass uint16
}

// parseHeader extracts DNS header fields from the first 12 bytes
func parseHeader(data []byte) DNSheader {
	if len(data) < 12 {
		return DNSheader{}
	}

	return DNSheader{
		ID:      binary.BigEndian.Uint16(data[0:2]),
		Flags:   binary.BigEndian.Uint16(data[2:4]),
		QDCount: binary.BigEndian.Uint16(data[4:6]),
		ANCount: binary.BigEndian.Uint16(data[6:8]),
		NSCount: binary.BigEndian.Uint16(data[8:10]),
		ARCount: binary.BigEndian.Uint16(data[10:12]),
	}
}

// buildResponseHeader creates a response header based on the request
func buildResponseHeader(requestHeader DNSheader) DNSheader {
	// Extract OPCODE from request flags (bits 11-14)
	opcode := (requestHeader.Flags >> 11) & 0x0F

	// Build response flags:
	// QR=1 (response), OPCODE from request, AA=0, TC=0, RD from request
	// RA=0, Z=0, RCODE=0 (no error)
	var flags uint16
	flags |= 1 << 15                        // QR = 1 (response)
	flags |= opcode << 11                   // OPCODE from request
	flags |= (requestHeader.Flags & 0x0100) // Copy RD (recursion desired)

	return DNSheader{
		ID:      requestHeader.ID,
		Flags:   flags,
		QDCount: requestHeader.QDCount,
		ANCount: 0, // No answers yet
		NSCount: 0,
		ARCount: 0,
	}
}

// encodeHeader converts a DNS header to bytes (12 bytes)
func encodeHeader(header DNSheader) []byte {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint16(buf[0:2], header.ID)
	binary.BigEndian.PutUint16(buf[2:4], header.Flags)
	binary.BigEndian.PutUint16(buf[4:6], header.QDCount)
	binary.BigEndian.PutUint16(buf[6:8], header.ANCount)
	binary.BigEndian.PutUint16(buf[8:10], header.NSCount)
	binary.BigEndian.PutUint16(buf[10:12], header.ARCount)
	return buf
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// TODO: Uncomment the code below to pass the first stage
	//
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053") // reason for resolveudpaddr is to create a udp address structure ex: &UDPAddr{IP: net.IPv4(127,0,0,1), Port: 2053}
	if err != nil {
		fmt.Println("Failed to resolve UDP address:", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return
	}
	defer udpConn.Close()

	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		fmt.Printf("Received %d bytes from %s\n", size, source)

		// Parse the request header
		requestHeader := parseHeader(buf[:size])
		fmt.Printf("Request ID: %d, Flags: 0x%04x, Questions: %d\n",
			requestHeader.ID, requestHeader.Flags, requestHeader.QDCount)

		// Build response header
		responseHeader := buildResponseHeader(requestHeader)

		// Encode to bytes
		response := encodeHeader(responseHeader)

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
