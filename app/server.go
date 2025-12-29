package main

import (
	"fmt"
	"net"

	"github.com/codecrafters-io/dns-server-starter-go/app/dns"
)

// DNSServer handles DNS server operations
type DNSServer struct {
	conn     *net.UDPConn
	resolver string
}

// NewDNSServer creates a new DNS server instance
func NewDNSServer(addr, resolver string) (*DNSServer, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	return &DNSServer{conn: conn, resolver: resolver}, nil
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

	// If resolver is set, forward the query
	if s.resolver != "" {
		return s.forwardQuery(&request)
	}

	// Build response (for non-forwarding mode)
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

// forwardQuery forwards a DNS query to the resolver and returns the response
func (s *DNSServer) forwardQuery(request *dns.DNSMessage) ([]byte, error) {
	// If multiple questions, split them and merge responses
	if len(request.Questions) > 1 {
		return s.forwardMultipleQuestions(request)
	}

	// Single question - forward directly
	return s.forwardSingleQuery(request)
}

// forwardSingleQuery forwards a single query to the resolver
func (s *DNSServer) forwardSingleQuery(request *dns.DNSMessage) ([]byte, error) {
	// Connect to resolver
	resolverAddr, err := net.ResolveUDPAddr("udp", s.resolver)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve resolver address: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, resolverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to resolver: %v", err)
	}
	defer conn.Close()

	// Send query to resolver
	queryBytes := request.Encode()
	_, err = conn.Write(queryBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to send query to resolver: %v", err)
	}

	// Read response from resolver
	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from resolver: %v", err)
	}

	return buf[:n], nil
}

// forwardMultipleQuestions splits multiple questions into separate queries and merges responses
func (s *DNSServer) forwardMultipleQuestions(request *dns.DNSMessage) ([]byte, error) {
	originalID := request.Header.ID
	var allAnswers []dns.DNSAnswer

	// Process each question separately
	for _, question := range request.Questions {
		// Create a new message with single question
		singleQuery := dns.DNSMessage{
			Header: dns.DNSHeader{
				ID:      request.Header.ID,
				Flags:   request.Header.Flags,
				QDCount: 1,
				ANCount: 0,
				NSCount: 0,
				ARCount: 0,
			},
			Questions: []dns.Question{question},
		}

		// Forward the single query
		responseBytes, err := s.forwardSingleQuery(&singleQuery)
		if err != nil {
			fmt.Printf("Error forwarding question: %v\n", err)
			continue
		}

		// Parse the response
		var response dns.DNSMessage
		if err := response.ParseComplete(responseBytes); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			continue
		}

		// Collect answers
		allAnswers = append(allAnswers, response.Answers...)
	}

	// Build merged response
	mergedResponse := dns.DNSMessage{
		Header: dns.DNSHeader{
			ID:      originalID,
			Flags:   request.Header.BuildResponse().Flags,
			QDCount: uint16(len(request.Questions)),
			ANCount: uint16(len(allAnswers)),
			NSCount: 0,
			ARCount: 0,
		},
		Questions: request.Questions,
		Answers:   allAnswers,
	}

	return mergedResponse.Encode(), nil
}
