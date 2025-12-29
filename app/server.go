package main

import (
	"fmt"
	"net"

	"github.com/codecrafters-io/dns-server-starter-go/app/dns"
)

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

// HandleQuery processes a DNS query and returns the response
func (s *DNSServer) HandleQuery(data []byte) ([]byte, error) {
	// Parse the request
	var request dns.DNSMessage
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
