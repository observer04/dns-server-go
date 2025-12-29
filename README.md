[![progress-banner](https://backend.codecrafters.io/progress/dns-server/e01fbaf6-73cd-4f89-973b-4c8ca8c67070)](https://app.codecrafters.io/users/codecrafters-bot?r=2qF)

# DNS Server in Go

A fully functional DNS server implementation in Go that supports both standalone operation and forwarding mode. Built as part of the CodeCrafters DNS Server challenge.

## Table of Contents
- [Overview](#overview)
- [Project Structure](#project-structure)
- [DNS Protocol Primer](#dns-protocol-primer)
- [Architecture](#architecture)
- [Component Details](#component-details)
- [Running the Server](#running-the-server)
- [Testing](#testing)
- [Implementation Notes](#implementation-notes)

## Overview

This DNS server can operate in two modes:
1. **Standalone Mode**: Responds to DNS queries with dummy answers (IP: 8.8.8.8)
2. **Forwarding Mode**: Acts as a DNS forwarder, relaying queries to an upstream resolver

### Features
✅ Full DNS message parsing and encoding  
✅ Support for DNS compression (pointer-based name encoding)  
✅ Query forwarding with upstream resolver  
✅ Multiple question handling (splits and merges)  
✅ Proper OPCODE and RCODE handling  
✅ Support for A record queries  

## Project Structure

```
.
├── app/
│   ├── main.go              # Entry point, CLI argument parsing
│   ├── server.go            # UDP server and query handling logic
│   └── dns/                 # DNS protocol implementation
│       ├── message.go       # Complete DNS message structure
│       ├── header.go        # DNS header (12 bytes)
│       ├── question.go      # Question section + name decoding
│       ├── answer.go        # Answer section + parsing
├── go.mod                   # Go module definition
├── your_program.sh          # Wrapper script for codecrafters
└── README.md                # This file
```

## DNS Protocol Primer

### Message Structure
A DNS message consists of:

```
+---------------------------+
|         Header            |  12 bytes (fixed)
+---------------------------+
|        Questions          |  Variable length
+---------------------------+
|         Answers           |  Variable length
+---------------------------+
|       Authority           |  Variable length (not implemented)
+---------------------------+
|       Additional          |  Variable length (not implemented)
+---------------------------+
```

### Header Format (12 bytes)
```
    0  1  2  3  4  5  6  7  8  9  10 11 12 13 14 15
  +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
  |                      ID                       |
  +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
  |QR|   Opcode  |AA|TC|RD|RA|   Z    |   RCODE   |
  +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
  |                    QDCOUNT                    |
  +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
  |                    ANCOUNT                    |
  +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
  |                    NSCOUNT                    |
  +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
  |                    ARCOUNT                    |
  +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
```

### Name Compression
DNS uses compression to reduce message size. Domain names can be:
- **Label sequences**: `\x06google\x03com\x00` (length-prefixed labels)
- **Pointers**: `\xC0\x0C` (2 bytes, points to offset 12 in the message)
- **Mixed**: Labels followed by a pointer

## Architecture

### Request Flow

#### Standalone Mode:
```
Client → UDP:2053 → Parse Request → Build Dummy Response → Send Response
```

#### Forwarding Mode:
```
Client → UDP:2053 → Parse Request → Forward to Resolver → Parse Resolver Response → Send Response
                                ↓
                          (Multiple Questions?)
                                ↓
                    Split → Forward Each → Merge Answers
```

### Key Design Decisions

1. **Pointer Handling**: `DecodeName()` tracks whether it jumped via pointer to correctly calculate bytes consumed
2. **Multiple Questions**: Resolver only accepts 1 question/query, so we split and merge
3. **ID Preservation**: Original packet ID is maintained throughout forwarding
4. **Error Handling**: Invalid OPCODEs return RCODE=4 (Not Implemented)

## Component Details

### 1. `main.go` - Entry Point
```go
// Responsibilities:
- Parse --resolver flag
- Initialize DNSServer with configuration
- Start the server
```

**Key Code:**
```go
resolverAddr := flag.String("resolver", "", "DNS resolver address (ip:port)")
server, err := NewDNSServer("127.0.0.1:2053", *resolverAddr)
```

### 2. `server.go` - Server Logic
```go
// Responsibilities:
- UDP socket management
- Request routing (forward vs standalone)
- Connection to upstream resolver
- Multiple question splitting/merging
```

**Key Methods:**
- `HandleQuery()`: Routes to forwarding or local response
- `forwardQuery()`: Decides single vs multiple question handling
- `forwardSingleQuery()`: Forwards one query to resolver
- `forwardMultipleQuestions()`: Splits, forwards, and merges responses

### 3. `dns/header.go` - Header Section
```go
// Responsibilities:
- Parse 12-byte header from bytes
- Build response headers with proper flags
- Handle OPCODE and RCODE logic
```

**Key Fields:**
- `ID`: Packet identifier (must match in response)
- `Flags`: QR, OPCODE, AA, TC, RD, RA, RCODE
- `QDCount`, `ANCount`, `NSCount`, `ARCount`: Section counts

**Flag Handling:**
```go
// Response flags construction:
flags |= 1 << 15            // QR = 1 (response)
flags |= opcode << 11       // Copy OPCODE from request
flags |= (h.Flags & 0x0100) // Copy RD bit
flags |= rcode              // Set RCODE
```

### 4. `dns/question.go` - Question Section
```go
// Responsibilities:
- Parse question (name, type, class)
- Decode compressed domain names
- Handle pointer following
```

**Key Function: `DecodeName()`**
```go
// Handles three scenarios:
1. Regular labels: \x03www\x06google\x03com\x00
2. Pointers: \xC0\x0C (jumps to offset 12)
3. Mixed: Labels + pointer at end

// Critical: Tracks "jumped" state to return correct byte count
```

**Pointer Detection:**
```go
if b&0xC0 == 0xC0 {  // Top 2 bits = 11
    ptr := binary.BigEndian.Uint16(data[currentOffset:currentOffset+2])
    newOffset := int(ptr & 0x3FFF)  // Lower 14 bits = offset
}
```

### 5. `dns/answer.go` - Answer Section
```go
// Responsibilities:
- Parse answer records from resolver responses
- Encode answers for client responses
- Handle RData (IP addresses, etc.)
```

**Answer Format:**
```
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                     NAME                      |  Variable (compressed)
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                     TYPE                      |  2 bytes
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                     CLASS                     |  2 bytes
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                      TTL                      |  4 bytes
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                   RDLENGTH                    |  2 bytes
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                     RDATA                     |  Variable (RDLENGTH bytes)
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
```

### 6. `dns/message.go` - Complete Message
```go
// Responsibilities:
- Coordinate parsing of all sections
- Build complete responses
- Encode messages to bytes
```

**Two Parse Methods:**
- `Parse()`: Header + Questions only (for incoming requests)
- `ParseComplete()`: Header + Questions + Answers (for resolver responses)

## Running the Server

### Build
```bash
go build -o dns-server ./app/
```

### Standalone Mode
```bash
./dns-server
# Listens on 127.0.0.1:2053
# Returns 8.8.8.8 for all A record queries
```

### Forwarding Mode
```bash
./dns-server --resolver 8.8.8.8:53
# Forwards all queries to Google's DNS server
```

### Using the Wrapper Script
```bash
./your_program.sh --resolver 1.1.1.1:53
```

## Testing

### Manual Testing with dig
```bash
# Terminal 1: Start server
./dns-server --resolver 8.8.8.8:53

# Terminal 2: Send query
dig @127.0.0.1 -p 2053 google.com
```

### Testing Multiple Questions
```bash
# The server automatically splits multiple questions
dig @127.0.0.1 -p 2053 google.com codecrafters.io
```

### CodeCrafters Testing
```bash
codecrafters test
codecrafters submit
```

## Implementation Notes

### Critical Edge Cases Handled

1. **Pointer Loops**: Max 1000 iterations prevents infinite loops in `DecodeName()`
2. **Buffer Overflows**: All parsing checks bounds before reading
3. **Multiple Questions**: Resolver limitation requires splitting
4. **ID Preservation**: Original request ID must match response ID
5. **OPCODE Validation**: Non-zero OPCODEs return RCODE=4

### Performance Considerations

- **UDP Buffer**: 512 bytes (DNS standard max for UDP)
- **No Caching**: Each query is forwarded fresh
- **Synchronous**: One query processed at a time
- **No Connection Pooling**: New UDP connection per query

### Limitations & Future Enhancements

**Current Limitations:**
- Only A record type tested/guaranteed
- Authority and Additional sections not implemented
- No DNSSEC support
- No TCP support for large responses
- No query caching

**Possible Enhancements:**
- [ ] Add caching layer with TTL expiration
- [ ] Support more record types (AAAA, MX, CNAME, etc.)
- [ ] Implement TCP fallback for truncated responses
- [ ] Add concurrent query handling (goroutines)
- [ ] Connection pooling for upstream resolver
- [ ] Metrics and logging improvements
- [ ] Configuration file support

### Debugging Tips

**Enable Verbose Logging:**
- The code already prints request IDs, flags, and question counts
- Add more `fmt.Printf()` statements in parsing functions for debugging

**Common Issues:**
```go
// Issue: Wrong byte count returned from DecodeName
// Fix: Ensure 'jumped' flag properly tracks pointer jumps

// Issue: Answer section parsing fails
// Fix: Check that pointer handling works in answer names

// Issue: Multiple questions not merged
// Fix: Verify each sub-query completes before merging
```

**Useful dig flags:**
```bash
dig +noedns    # Disable EDNS (simpler to debug)
dig +short     # Just show the answer
dig +trace     # Show full resolution path
```

## Resources

- [RFC 1035](https://www.rfc-editor.org/rfc/rfc1035) - Domain Names Implementation
- [DNS Message Format](https://www2.cs.duke.edu/courses/fall16/compsci356/DNS/DNS-primer.pdf)
- [CodeCrafters DNS Challenge](https://app.codecrafters.io/courses/dns-server)

---

**Author Notes**: This implementation prioritizes clarity and correctness over performance. Each component is well-isolated, making it easy to understand, test, and modify individual parts of the DNS protocol handling.

