package main

import (
	"fmt"
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	// Create and start DNS server
	server, err := NewDNSServer("127.0.0.1:2053")
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}

	fmt.Printf("DNS server listening on %s\n", server.conn.LocalAddr().String())
	if err := server.Run(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
