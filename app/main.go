package main

import (
	"flag"
	"fmt"
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	// Parse command line arguments
	resolverAddr := flag.String("resolver", "", "DNS resolver address (ip:port)")
	flag.Parse()

	// Create and start DNS server
	server, err := NewDNSServer("127.0.0.1:2053", *resolverAddr)
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}

	if *resolverAddr != "" {
		fmt.Printf("Forwarding queries to resolver: %s\n", *resolverAddr)
	}

	fmt.Printf("DNS server listening on %s\n", server.conn.LocalAddr().String())
	if err := server.Run(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
