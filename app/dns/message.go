package dns

// DNSMessage represents a complete DNS message
type DNSMessage struct {
	Header    DNSHeader
	Questions []Question
	Answers   []DNSAnswer
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

// ParseComplete parses a full DNS message including answers
func (msg *DNSMessage) ParseComplete(data []byte) error {
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

	// Parse answers
	msg.Answers = make([]DNSAnswer, 0, msg.Header.ANCount)
	for i := uint16(0); i < msg.Header.ANCount; i++ {
		var a DNSAnswer
		bytesRead, err := a.Parse(data, offset)
		if err != nil {
			return err
		}
		msg.Answers = append(msg.Answers, a)
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
		Answers:   make([]DNSAnswer, len(msg.Questions)),
	}

	// Copy questions and add dummy A records for each
	for i, q := range msg.Questions {
		response.Questions[i] = q
		response.Answers[i] = DNSAnswer{
			Name:     q.QName,
			Type:     q.QType, // Use the requested type
			Class:    q.QClass,
			TTL:      60,
			RDLength: 4,
			RData:    []byte{8, 8, 8, 8}, // Dummy IP
		}
	}

	return response
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
